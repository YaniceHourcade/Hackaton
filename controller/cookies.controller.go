package controllers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
)

type Cookie struct {
	Name  string
	Value string

	Path       string
	Domain     string
	Expires    time.Time
	RawExpires string

	MaxAge   int
	Secure   bool
	HttpOnly bool
	SameSite http.SameSite
	Raw      string
	Unparsed []string
}

var passphrase = []byte("CatHubEstLeMeilleurHub")

// Fonction pour créer un JWT
func createJWT(userID int) (string, error) {
	claims := jwt.MapClaims{}

	claims["id"] = userID
	claims["exp"] = time.Now().Add(time.Hour * 24 * 7).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(passphrase)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// Fonction pour lire les donnée d'un JWT
func readJWT(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("méthode de signature invalide : %v", token.Header["alg"])
		}
		return passphrase, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	} else {
		return nil, fmt.Errorf("token invalide")
	}
}

// Fonction pour transformer un cookie de session en user ID
func GetUserIDFromCookie(cookie *http.Cookie) (int, error) {
	claims, err := readJWT(cookie.Value)
	if err != nil {
		return -1, err
	}

	userIDFloat, ok := claims["id"].(float64)
	if !ok {
		return -1, fmt.Errorf("ID invalide dans les revendications du JWT")
	}
	userID := int(userIDFloat)

	return userID, nil
}
