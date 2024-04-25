package controllers

import (
	models "choppetoncolis/model"
	"database/sql"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	_ "image/jpeg"
	_ "image/png"

	_ "github.com/mattn/go-sqlite3"
)

// ===== STRUCTS =====
type PanelPage struct {
	AllColis      []*models.Colis
	TargetedColis *models.Colis
	AllUsers      []*models.User
	TargetedUser  *models.User

	NewUserCreation bool
	Email           string
	EmailMessage    string
	PasswordMessage string

	Error         error
	GlobalMessage string
}

// ===================

func PanelHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, sessionErr := r.Cookie("session")
		if sessionErr != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		currentUserID, err := GetUserIDFromCookie(sessionCookie)
		if err != nil {
			http.Redirect(w, r, "/logout", http.StatusFound)
			return
		}
		currentUser, err := models.GetUser(db, currentUserID)
		if err != nil {
			http.Redirect(w, r, "/logout", http.StatusFound)
			return
		} else if !currentUser.IsAdmin {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		p := &PanelPage{}

		//=== COLIS ===
		p.AllColis, p.Error = models.GetAllColis(db)
		path := r.URL.Path

		// Affichage d'un colis spécifique
		colisID := strings.TrimPrefix(path, "/panel/colis/")
		if colisID != "" {
			p.TargetedColis, p.Error = models.GetColis(db, colisID)
		}

		//=== USERS ===
		p.AllUsers, p.Error = models.GetAllUsers(db)

		// Affichage d'un user spécifique
		userID := strings.TrimPrefix(path, "/panel/user/")
		userIDint, _ := strconv.Atoi(userID)
		if userID != "" {
			if userID == "new" {
				p.NewUserCreation = true
			} else {
				p.TargetedUser, p.Error = models.GetUser(db, userIDint)
			}
		}

		t, _ := template.ParseFiles("./view/panel.html")
		t.Execute(w, p)
	}
}
