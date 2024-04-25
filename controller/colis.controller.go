package controllers

import (
	models "choppetoncolis/model"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"

	_ "image/jpeg"
	_ "image/png"

	_ "github.com/mattn/go-sqlite3"
	"github.com/skip2/go-qrcode"
)

func GenerateQRCodeHandler(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")

	qr, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		http.Error(w, "Impossible de générer le QR code", http.StatusInternalServerError)
		return
	}

	pngData, err := qr.PNG(256)
	if err != nil {
		http.Error(w, "Impossible de générer l'image PNG", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	if _, err := w.Write(pngData); err != nil {
		log.Println("Erreur lors de l'écriture des données de l'image:", err)
	}
}

func NewColisHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, sessionErr := r.Cookie("session")
		if sessionErr != nil {
			http.Redirect(w, r, "/login", http.StatusNotFound)
			return
		} else if !IsUserAdminFromCookie(db, r) {
			http.Redirect(w, r, "/logout", http.StatusNotFound)
			return
		}

		if r.Method == http.MethodPost {
			newColisID, err := models.AddColis(db)
			if err != nil {
				http.Redirect(w, r, "/panel", http.StatusFound)
				return
			}

			http.Redirect(w, r, "/panel/colis/"+newColisID, http.StatusFound)
			return
		} else {
			http.Redirect(w, r, "/panel", http.StatusFound)
			return
		}
	}
}

func ValidColis(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		isCurrentUserAdmin := IsUserAdminFromCookie(db, r)

		colisID, transitID, extractErr := extractIDsFromURL(r)
		if extractErr != nil {
			http.Error(w, extractErr.Error(), http.StatusBadRequest)
			return
		}

		action := r.URL.Query().Get("action")

		newTransitID, err := models.SetColisStep(db, colisID, transitID, action)
		if err != nil {
			// Si l'utilisateur est connecté et qu'il y'a une erreur
			if isCurrentUserAdmin {
				http.Redirect(w, r, "/panel/colis/"+colisID, http.StatusNotFound)
				return
			}

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if isCurrentUserAdmin {
			http.Redirect(w, r, "/panel/colis/"+colisID, http.StatusFound)
			return
		}
		http.Redirect(w, r, "/qr?url="+(colisID)+"/"+newTransitID, http.StatusFound)
		return
	}
}

func extractIDsFromURL(r *http.Request) (string, string, error) {
	// Utilisation d'une expression régulière pour extraire les IDs du colis et du transit du chemin de l'URL
	re := regexp.MustCompile(`/colis/([^/]+)/([^/]+)`)
	matches := re.FindStringSubmatch(r.URL.Path)
	if len(matches) != 3 {
		return "", "", errors.New("impossible de trouver les IDs dans l'URL")
	}

	colisID := matches[1]
	transitID := matches[2]

	return colisID, transitID, nil
}

func DeleteColisHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, sessionErr := r.Cookie("session")
		if sessionErr != nil {
			http.Redirect(w, r, "/login", http.StatusNotFound)
			return
		}
		currentUserID, err := GetUserIDFromCookie(sessionCookie)
		if err != nil {
			http.Redirect(w, r, "/logout", http.StatusNotFound)
			return
		}
		currentUser, err := models.GetUser(db, currentUserID)
		if err != nil {
			http.Redirect(w, r, "/logout", http.StatusNotFound)
			return
		} else if !currentUser.IsAdmin {
			http.Redirect(w, r, "/", http.StatusNotFound)
			return
		}

		if r.Method == http.MethodPost {
			path := r.URL.Path
			cid := strings.TrimPrefix(path, "/panel/colis/delete/")
			if cid != "" {
				err := models.DeleteColis(db, cid)
				if err == nil {
					http.Redirect(w, r, "/panel", http.StatusFound)
					return
				} else {
					http.Redirect(w, r, "/panel/colis/"+cid, http.StatusFound)
					return
				}
			}
		}

		http.Redirect(w, r, "/panel", http.StatusFound)
		return
	}
}
