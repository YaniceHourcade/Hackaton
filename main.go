package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	controllers "choppetoncolis/controller"
	models "choppetoncolis/model"
)

// ===== STRUCTS =====
type Page struct {
	Title       string
	IsAuth      bool
	CurrentUser *models.User
	ColisData   *models.Colis
	Error       error
}

//===================

//===== Pages HTML =====

// handler pour la page d'accueil
func handler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			NotFound(w, r)
			return
		}

		currentUser := &models.User{}

		sessionCookie, sessionErr := r.Cookie("session")
		if sessionErr != nil {
			sessionCookie = &http.Cookie{Value: ""}
		} else {
			currentUserID, err := controllers.GetUserIDFromCookie(sessionCookie)
			if err != nil {
				currentUser = &models.User{}
			} else {
				currentUser, err = models.GetUser(db, currentUserID)
				if err != nil {
					currentUser = &models.User{}
				}
			}
		}

		p := Page{
			Title:       "Accueil",
			IsAuth:      (sessionErr == nil),
			CurrentUser: currentUser,
		}

		colisURL := r.URL.Query().Get("colis")
		if colisURL != "" {
			if len(colisURL) == 8 {
				colis, err := models.GetColisFromPID(db, colisURL[:8])
				if err != nil {
					log.Println("Erreur lors de la récupération du colis :", err)
					p.Error = fmt.Errorf("Colis introuvable")
				} else {
					p.ColisData = colis
					p.Title = strings.ToUpper(colisURL)
				}
			} else {
				p.Error = fmt.Errorf("Code colis invalide")
			}
		}

		t, err := template.ParseFiles("./view/index.html")
		if err != nil {
			http.Error(w, "Erreur lors de la lecture du template HTML", http.StatusInternalServerError)
			return
		}
		t.Execute(w, p)
	}
}

// Handler Page 404
func NotFound(w http.ResponseWriter, r *http.Request) {
	p := Page{
		Title: "Page non trouvée",
	}
	t, err := template.ParseFiles("./view/404.html")
	if err != nil {
		http.Error(w, "Erreur lors de la lecture de la page 404", http.StatusInternalServerError)
		fmt.Println(err)
		return
	}
	t.Execute(w, p)
}

//======================

//===== API =====

func main() {
	db, err := sql.Open("sqlite3", "./database.db")
	if err != nil {
		fmt.Println("Error opening database:", err)
		return
	}
	defer db.Close()

	//=== Initialisation des tables colis et users ===
	if err := models.InitializeColisDB(db); err != nil {
		log.Fatal("Erreur lors de l'initialisation de la base de données colis:", err)
	}
	if err := models.InitializeUsersDB(db); err != nil {
		log.Fatal("Erreur lors de l'initialisation de la base de données users:", err)
	}
	//======

	//=== Serveur de fichier statiques ===
	fs := http.FileServer(http.Dir("./static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	//======

	//=== Routes ===
	http.HandleFunc("/colis/", controllers.ValidColis(db))
	http.HandleFunc("/panel", controllers.PanelHandler(db))
	http.HandleFunc("/panel/colis/", controllers.PanelHandler(db))
	http.HandleFunc("/panel/colis/new", controllers.NewColisHandler(db))
	http.HandleFunc("/panel/colis/delete/", controllers.DeleteColisHandler(db))

	http.HandleFunc("/panel/user/delete/", controllers.DeleteUserHandler(db))
	http.HandleFunc("/panel/user/edit/", controllers.EditUserHandler(db))
	http.HandleFunc("/panel/user/", controllers.PanelHandler(db))

	http.HandleFunc("/login", controllers.LoginHandler(db))
	http.HandleFunc("/register", controllers.RegisterHandler(db))
	http.HandleFunc("/logout", controllers.LogoutHandler)

	http.HandleFunc("/qr", controllers.GenerateQRCodeHandler)
	http.HandleFunc("/404", NotFound)
	http.HandleFunc("/", handler(db))
	//======

	fmt.Println("Serveur web lancé sur http://127.0.0.1:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
