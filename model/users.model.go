package models

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const UsersImageUploadPath = "./uploads/users"
const DefaultProfileImageFilename = "profile.jpg"

// ===== BASICS =====
func InitializeUsersDB(db *sql.DB) error {
	checkTableSQL := `SELECT count(*) FROM sqlite_master WHERE type='table' AND name='users';`
	var tableExists int
	err := db.QueryRow(checkTableSQL).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("erreur lors de la vérification de l'existence de la table users: %v", err)
	}

	if tableExists == 0 {
		createTableUsersSQL := `
			CREATE TABLE IF NOT EXISTS users (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				email VARCHAR(320) NOT NULL UNIQUE,
				password VARCHAR(255) NOT NULL,
				salt VARCHAR(16) NOT NULL,
				admin BOOL DEFAULT FALSE,
				createdAt INT NOT NULL
			);
		`
		_, err := db.Exec(createTableUsersSQL)
		if err != nil {
			return fmt.Errorf("erreur lors de la création de la table des utilisateurs: %v", err)
		}

		_, err = RegisterUser(db, "admin@admin.com", "Admin.1234", true)
		if err != nil {
			return fmt.Errorf("erreur lors de la création de l'utilisateur par défaut: %v", err)
		}
	}

	return nil
}

//==========

// ===== CONST =====
const EMAIL_MAX_LENGTH = 320
const EMAIL_REGEX = `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`

const PASSWORD_MAX_LENGTH = 255
const PASSWORD_REGEX = `^[A-Za-z\d!@#$&*./]{8,}$`

//==========

// ===== STRUCTS =====
type User struct {
	ID                 int
	Email              string
	CreatedAt          int
	CreatedAtFormatted string
	IsAdmin            bool
}

//==========

//===== UTILS =====

// Permet de récupérer le timestamp actuel
func getCurrentTimestamp() int64 {
	location, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		fmt.Println("Erreur lors du chargement du fuseau horaire:", err)
		return 0
	}

	timeInParis := time.Now().In(location)
	timestamp := timeInParis.Unix()

	return timestamp
}

// Permet de hash un string en sha-256 via un salt
func hashPassword(salt string, password string) string {
	hasher := sha256.New()
	hasher.Write([]byte(salt + password))
	hashedBytes := hasher.Sum(nil)
	hashedString := fmt.Sprintf("%x", hashedBytes)
	return hashedString
}

// Permet de générer un salt
func generateSalt() (string, error) {
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}
	saltString := hex.EncodeToString(salt)
	return saltString, nil
}

// Permet de vérifier si une adresse email est valide par rapport à son regex
func ValidateEmail(email string) error {
	email = strings.ToLower(email)
	if email == "" {
		return fmt.Errorf("une adresse email doit être spécifiée")
	}
	if len(email) > EMAIL_MAX_LENGTH {
		return fmt.Errorf("l'adresse email doit faire moins de 320 caractères")
	}
	emailRegex, err := regexp.Compile(EMAIL_REGEX)
	if err != nil {
		return fmt.Errorf("erreur de validation de l'adresse email: %v", err)
	}
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("l'adresse email doit être valide")
	}
	return nil
}

// Permet de vérifier si un mot de passe est valide par rapprot à son regex
func ValidatePassword(password string) error {
	if password == "" {
		return fmt.Errorf("un mot de passe doit être spécifié")
	}
	if len(password) > PASSWORD_MAX_LENGTH {
		return fmt.Errorf("le mot de passe ne doit pas dépasser les 255 caractères")
	}

	hasUppercase := strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	hasLowercase := strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz")
	hasDigit := strings.ContainsAny(password, "0123456789")
	hasSpecial := strings.ContainsAny(password, "!@#$&*./")

	if !hasUppercase || !hasLowercase || !hasDigit || !hasSpecial {
		return fmt.Errorf("le mot de passe doit avoir au moins 1 majuscule, 1 minuscule, 1 chiffre et 1 caractère spécial (!@#$&*./)")
	}
	passwordRegex := regexp.MustCompile(PASSWORD_REGEX)
	if !passwordRegex.MatchString(password) {
		return fmt.Errorf("le mot de passe doit avoir au moins 1 majuscule, 1 minuscule, 1 chiffre et 1 caractère spécial (!@#$&*./)")
	}

	return nil
}

//==========

// ===== Messages d'erreurs =====
var (
	ErrInvalidEmail    = errors.New("l'adresse email doit être valide")
	ErrInvalidPassword = errors.New("le mot de passe doit avoir au moins 1 majuscule, 1 minuscule, 1 chiffre et 1 caractère spécial")
	ErrUserExists      = errors.New("cette adresse email est déjà utilisée")

	ErrInvalidAuth = errors.New("erreur d'authentification")
)

//==========

// ===== FONCTIONS =====
func GetAllUsers(db *sql.DB) ([]*User, error) {
	var allUsers []*User

	rowsUser, err := db.Query("SELECT id, email, admin, createdAt FROM users")
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la vérification de l'existence de l'utilisateur: %v", err)
	}
	defer rowsUser.Close()

	for rowsUser.Next() {
		var id int
		var email string
		var admin bool
		var createdAt int
		err := rowsUser.Scan(&id, &email, &admin, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("erreur lors de la récupération de l'utilisateur: %v", err)
		} else {
			CreatedAtFormatted := time.Unix(int64(createdAt), 0).Format("15:04 - 02/01/2006")

			user := &User{
				ID:                 id,
				Email:              email,
				CreatedAt:          createdAt,
				CreatedAtFormatted: CreatedAtFormatted,
				IsAdmin:            admin,
			}

			allUsers = append(allUsers, user)
		}
	}
	return allUsers, nil
}

func GetUser(db *sql.DB, id int) (*User, error) {

	rowsUser, err := db.Query("SELECT id, email, admin, createdAt FROM users WHERE id = ?", id)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la vérification de l'existence de l'utilisateur: %v", err)
	}
	defer rowsUser.Close()

	if rowsUser.Next() {
		var id int
		var email string
		var admin bool
		var createdAt int

		err := rowsUser.Scan(&id, &email, &admin, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("erreur lors de la récupération de l'utilisateur: %v", err)
		} else {
			CreatedAtFormatted := time.Unix(int64(createdAt), 0).Format("15:04 - 02/01/2006")

			user := &User{
				ID:                 id,
				Email:              email,
				CreatedAt:          createdAt,
				CreatedAtFormatted: CreatedAtFormatted,
				IsAdmin:            admin,
			}

			return user, nil
		}
	}
	return nil, nil
}

// Fonction pour enregister un utilisateur dans la base de donnée
func RegisterUser(db *sql.DB, email, password string, setAdmin bool) (int, error) {
	if err := ValidateEmail(email); err != nil {
		return 0, ErrInvalidEmail
	}
	if err := ValidatePassword(password); err != nil {
		return 0, ErrInvalidPassword
	}

	var count int
	row := db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", email)
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("erreur lors de la vérification de l'existence de l'utilisateur: %v", err)
	}
	if count > 0 {
		return 0, ErrUserExists
	}

	salt, err := generateSalt()
	if err != nil {
		return 0, fmt.Errorf("erreur lors de la génération du sel: %v", err)
	}
	hashedPassword := hashPassword(salt, password)

	result, err := db.Exec("INSERT INTO users (email, password, salt, admin, createdAt) VALUES (?, ?, ?, ?, ?)", email, hashedPassword, salt, setAdmin, getCurrentTimestamp())
	if err != nil {
		return 0, fmt.Errorf("erreur lors de l'insertion de l'utilisateur dans la base de données: %v", err)
	}

	lastInsertID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("erreur lors de la récupération de l'ID du dernier insert: %v", err)
	}

	return int(lastInsertID), nil
}

// Fonction pour vérifier le login d'un utilisateur
func LoginUser(db *sql.DB, email, password string) (int, error) {
	email = strings.ToLower(email)

	userId := -1

	if err := ValidateEmail(email); err != nil {
		return userId, ErrInvalidEmail
	}

	rowsEmail, err := db.Query("SELECT id, salt, password FROM users WHERE email = ?", email)
	if err != nil {
		return userId, fmt.Errorf("erreur lors de la vérification de l'existence de l'utilisateur: %v", err)
	}
	defer rowsEmail.Close()

	if rowsEmail.Next() {
		var salt string
		var goodPassword string
		err := rowsEmail.Scan(&userId, &salt, &goodPassword)
		if err != nil {
			return userId, fmt.Errorf("erreur lors de la récupération des données de l'utilisateur: %v", err)
		} else {
			hashedPassword := hashPassword(salt, password)
			if hashedPassword != goodPassword {
				return userId, ErrInvalidAuth
			}
		}
	} else {
		return userId, ErrInvalidAuth
	}

	return userId, nil
}

func DeleteUser(db *sql.DB, id int) error {
	allUsers, err := GetAllUsers(db)

	if err != nil {
		return fmt.Errorf("Une erreur a eu lieu")
	}
	if len(allUsers) <= 1 {
		return fmt.Errorf("Impossbile de supprimer tout les utilisateurs !")
	}

	_, err = GetUser(db, id)
	if err == nil {
		db.Exec("DELETE FROM users WHERE id = ?", id)
	}
	return nil
}

//==========
