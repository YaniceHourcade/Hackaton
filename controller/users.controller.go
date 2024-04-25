package controllers

import (
	models "choppetoncolis/model"
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "image/jpeg"
	_ "image/png"

	_ "github.com/mattn/go-sqlite3"
)

// ===== STRUCTS =====
type AuthPage struct {
	Username string
	Email    string

	EmailMessage    string
	PasswordMessage string
	GlobalMessage   string
}

// ===================

func IsUserAdminFromCookie(db *sql.DB, r *http.Request) bool {
	sessionCookie, sessionErr := r.Cookie("session")
	if sessionErr != nil {
		return false
	}
	currentUserID, err := GetUserIDFromCookie(sessionCookie)
	if err != nil {
		return false
	}
	currentUser, err := models.GetUser(db, currentUserID)
	if err != nil {
		return false
	} else if !currentUser.IsAdmin {
		return false
	}
	return true
}

func LoginHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, sessionErr := r.Cookie("session")
		if sessionErr == nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		p := &AuthPage{}

		if r.Method == http.MethodPost {
			email := r.FormValue("email")
			password := r.FormValue("password")
			remember := r.FormValue("remember")

			p.Email = email

			var maxAge int
			if remember == "on" {
				maxAge = 7 * 24 * 3600
			} else {
				maxAge = 0
			}

			if userId, err := models.LoginUser(db, email, password); err != nil {
				switch err {
				case models.ErrInvalidAuth:
					p.GlobalMessage = err.Error()

				case models.ErrInvalidEmail:
					p.GlobalMessage = err.Error()
				default:
					p.GlobalMessage = "Une erreur a eu lieu"
				}
			} else {
				tokenString, err := createJWT(userId)
				if err != nil {
					fmt.Println("Erreur lors de la création du JWT:", err)
					return
				} else {
					cookie := http.Cookie{
						Name:     "session",
						Value:    tokenString,
						Path:     "/",
						MaxAge:   maxAge,
						HttpOnly: true,
						Secure:   false,
						SameSite: http.SameSiteNoneMode,
					}

					http.SetCookie(w, &cookie)
				}

				http.Redirect(w, r, "/panel", http.StatusFound)
				return
			}
		}

		t, _ := template.ParseFiles("./view/auth.html")
		t.Execute(w, p)
	}
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	_, sessionErr := r.Cookie("session")
	if sessionErr == nil {
		cookie := http.Cookie{
			Name:     "session",
			Value:    "",
			Path:     "/",
			Expires:  time.Unix(0, 0),
			HttpOnly: true,
			Secure:   false,
		}

		http.SetCookie(w, &cookie)
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func EditUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, sessionErr := r.Cookie("session")
		if sessionErr != nil {
			http.Redirect(w, r, "/login", http.StatusNotFound)
			return
		} else if !IsUserAdminFromCookie(db, r) {
			http.Redirect(w, r, "/logout", http.StatusNotFound)
			return
		}

		// Récupération de l'id de l'utilisater à edit
		path := r.URL.Path
		userID := strings.TrimPrefix(path, "/panel/user/edit/")

		if r.Method == http.MethodPost {
			if userID != "" {
				userIDint, _ := strconv.Atoi(userID)
				userTarget, err := models.GetUser(db, userIDint)
				if err == nil {
					r.ParseForm()
					adminValue := r.Form.Get("admin")
					isAdmin := false
					if adminValue == "true" {
						isAdmin = true
					}
					db.Exec("UPDATE users SET admin = ? WHERE id = ?", isAdmin, userTarget.ID)
				}
			}
		}

		http.Redirect(w, r, "/panel/user/"+userID, http.StatusFound)
	}
}

func RegisterHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, sessionErr := r.Cookie("session")
		if sessionErr != nil {
			http.Redirect(w, r, "/login", http.StatusNotFound)
			return
		} else if !IsUserAdminFromCookie(db, r) {
			http.Redirect(w, r, "/logout", http.StatusNotFound)
			return
		}

		p := &PanelPage{}

		if r.Method == http.MethodPost {
			r.ParseForm()
			email := r.FormValue("email")
			password := r.FormValue("password")

			p.Email = email

			setAdmin := false
			adminCheckboxValue := r.FormValue("admin")
			if adminCheckboxValue == "on" {
				setAdmin = true
			}

			_, err := models.RegisterUser(db, email, password, setAdmin)
			if err != nil {
				/*
					switch err {
					case models.ErrInvalidEmail:
						p.EmailMessage = err.Error()

					case models.ErrInvalidPassword:
						p.PasswordMessage = err.Error()

					case models.ErrUserExists:
						p.GlobalMessage = err.Error()
					default:
						fmt.Println(err.Error())
						p.GlobalMessage = "Une erreur a eu lieu"
					}
				*/
				http.Redirect(w, r, "/panel/user/new", http.StatusFound)
				return
			}
		}

		http.Redirect(w, r, "/panel/user", http.StatusFound)
	}
}

func DeleteUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, sessionErr := r.Cookie("session")
		if sessionErr != nil {
			http.Redirect(w, r, "/login", http.StatusNotFound)
			return
		} else if !IsUserAdminFromCookie(db, r) {
			http.Redirect(w, r, "/logout", http.StatusNotFound)
			return
		}

		currentUserID, _ := GetUserIDFromCookie(sessionCookie)

		// Récupération de l'id de l'utilisater à edit
		path := r.URL.Path
		userID := strings.TrimPrefix(path, "/panel/user/delete/")

		if r.Method == http.MethodPost {
			if userID != "" {
				userIDint, _ := strconv.Atoi(userID)
				if currentUserID != userIDint {
					models.DeleteUser(db, userIDint)
				}
			}
		}

		http.Redirect(w, r, "/panel/user/", http.StatusFound)
	}
}
