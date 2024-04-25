package models

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ===== BASICS =====
func InitializeColisDB(db *sql.DB) error {
	checkTableSQL := `SELECT count(*) FROM sqlite_master WHERE type='table' AND name='colis';`
	var tableExists int
	err := db.QueryRow(checkTableSQL).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("erreur lors de la vérification de l'existence de la table colis: %v", err)
	}

	if tableExists == 0 {
		createTableSQL := `
            CREATE TABLE colis (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                colisID VARCHAR(36) NOT NULL UNIQUE,
                transitID VARCHAR(36) NOT NULL UNIQUE,
                step INT DEFAULT 0,
				maxStep INT DEFAULT 5,
                checkPointsDate STRING NOT NULL
            );
        `
		_, err := db.Exec(createTableSQL)
		if err != nil {
			return fmt.Errorf("erreur lors de la création de la table colis: %v", err)
		}
	}

	return nil
}

//==========

// ===== STRUCTS =====
type Colis struct {
	ID              int
	PublicColisID   string
	ColisID         string
	TransitID       string
	Step            int
	MaxStep         int
	CheckPoints     []string
	CheckPointsDate []string
}

//==========

// ===== Messages d'erreurs =====
var (
	ErrColisUnknow = errors.New("Colis inconnu")
)

//==========

// ===== FONCTIONS =====
func checkPointsToTimeStamp(checkpoints string) ([]int, error) {
	checkPointDateFormatted := strings.Split(checkpoints, ",")

	var checkPointsInt []int
	var globalErr error

	for _, date := range checkPointDateFormatted {
		dateInt, err := strconv.Atoi(date)
		if err != nil {
			checkPointsInt = []int{}
			globalErr = fmt.Errorf("une erreur a eu lieu lors de la récupération des dates de checkpoint d'un colis: %v", err)
			break
		}
		checkPointsInt = append(checkPointsInt, dateInt)
	}

	return checkPointsInt, globalErr
}

func AddColis(db *sql.DB) (string, error) {
	colisID := uuid.New().String()
	transitID := uuid.New().String()
	checkPointsDate := strconv.Itoa(int(getCurrentTimestamp()))

	_, err := db.Exec("INSERT INTO colis (colisID, transitID, checkPointsDate) VALUES (?, ?, ?)", colisID, transitID, checkPointsDate)
	if err != nil {
		return "", fmt.Errorf("erreur lors de l'insertion du colis dans la base de données: %v", err)
	}

	return colisID, nil
}

func DeleteColis(db *sql.DB, cid string) error {
	_, err := db.Exec("DELETE FROM colis WHERE colisID = ?", cid)
	if err != nil {
		return fmt.Errorf("erreur lors de la suppression du colis dans la base de données: %v", err)
	}

	return nil
}

func GetColis(db *sql.DB, cid string) (*Colis, error) {
	var colis *Colis

	rowsColis, err := db.Query("SELECT id, colisID, transitID, step, maxStep, checkPointsDate FROM colis WHERE colisID = ?", cid)
	if err != nil {
		return colis, fmt.Errorf("erreur lors de la vérification de l'existence du colis: %v", err)
	}
	defer rowsColis.Close()

	if rowsColis.Next() {
		var id int
		var colisID string
		var transitID string
		var step int
		var maxStep int
		var checkPointDate string
		err := rowsColis.Scan(&id, &colisID, &transitID, &step, &maxStep, &checkPointDate)
		if err != nil {
			return colis, fmt.Errorf("erreur lors de la récupération du colis: %v", err)
		}

		var checkPointsInt []int
		checkPointsInt, _ = checkPointsToTimeStamp(checkPointDate)

		var checkPoints []string
		for _, checkpoint := range checkPointsInt {
			checkPoints = append(checkPoints, strconv.Itoa(checkpoint))
		}

		var checkPointsDate []string
		for _, date := range checkPointsInt {
			checkPointsDate = append(checkPointsDate, time.Unix(int64(date), 0).Format("02/01 15:04"))
		}

		colis = &Colis{
			ID:              id,
			PublicColisID:   colisID[:8],
			ColisID:         colisID,
			TransitID:       transitID,
			Step:            step,
			MaxStep:         maxStep,
			CheckPoints:     checkPoints,
			CheckPointsDate: checkPointsDate,
		}
	}

	return colis, nil
}

func GetAllColis(db *sql.DB) ([]*Colis, error) {
	allColis := []*Colis{}

	rowsColis, err := db.Query("SELECT id, colisID, transitID, step, maxStep, checkPointsDate FROM colis")
	if err != nil {
		return allColis, fmt.Errorf("erreur lors de la vérification de l'existence du colis: %v", err)
	}
	defer rowsColis.Close()

	for rowsColis.Next() {
		var id int
		var colisID string
		var transitID string
		var step int
		var maxStep int
		var checkPointDate string
		err := rowsColis.Scan(&id, &colisID, &transitID, &step, &maxStep, &checkPointDate)
		if err != nil {
			return allColis, fmt.Errorf("erreur lors de la récupération du colis: %v", err)
		}

		colis := &Colis{
			ID:            id,
			PublicColisID: colisID[:8],
			ColisID:       colisID,
			Step:          step,
			MaxStep:       maxStep,
		}

		allColis = append(allColis, colis)
	}

	return allColis, nil
}

// PID (Public ID) Au format xxxxxx -> 6 premiers caractères du CID
// Exemple: b6fde652-4ffc-45b5-9ca6-483a81871d84 => b6fde652
func GetColisFromPID(db *sql.DB, pid string) (*Colis, error) {
	var colis *Colis

	pid = strings.ToLower(pid)
	rowsColis, err := db.Query("SELECT step, maxStep, checkPointsDate FROM colis WHERE colisID LIKE ?", pid+"%")
	if err != nil {
		return colis, fmt.Errorf("erreur lors de la vérification de l'existence du colis: %v", err)
	}
	defer rowsColis.Close()

	if !rowsColis.Next() {
		return colis, fmt.Errorf("Aucun colis trouvé pour l'ID %s", pid)
	}

	var step int
	var maxStep int
	var checkPointDate string
	err = rowsColis.Scan(&step, &maxStep, &checkPointDate)
	if err != nil {
		return colis, fmt.Errorf("erreur lors de la récupération du colis: %v", err)
	}

	var checkPointsInt []int
	checkPointsInt, _ = checkPointsToTimeStamp(checkPointDate)

	var checkPoints []string
	for _, checkpoint := range checkPointsInt {
		checkPoints = append(checkPoints, strconv.Itoa(checkpoint))
	}

	var checkPointsDate []string
	for _, date := range checkPointsInt {
		checkPointsDate = append(checkPointsDate, time.Unix(int64(date), 0).Format("02/01 15:04"))
	}

	colis = &Colis{
		PublicColisID:   pid,
		Step:            step,
		MaxStep:         maxStep,
		CheckPoints:     checkPoints,
		CheckPointsDate: checkPointsDate,
	}

	return colis, nil
}

func SetColisStep(db *sql.DB, colisID, transitID, action string) (string, error) {
	colis, err := GetColis(db, colisID)
	if err != nil {
		return "", err
	}
	// Vérifier si le colis existe
	if colis == nil {
		return "", fmt.Errorf("colis non trouvé dans la base de données")
	}

	// Vérifier si le transitID correspond
	if colis.TransitID != transitID {
		return "", fmt.Errorf("Mauvais Transit ID de colis")
	}

	// Si l'action est "cancel", rétablir l'étape précédente
	if action == "cancel" {
		if colis.Step < 0 {
			return "", fmt.Errorf("Le colis est déjà à son point de départ")
		}

		_, err = db.Exec("UPDATE colis SET step = step - 1 WHERE colisID = ?", colisID)
		if err != nil {
			return "", fmt.Errorf("erreur lors de l'incrémentation de la valeur de step dans la base de données: %v", err)
		}

		newColisCheckpoints := strings.Join(colis.CheckPoints[:len(colis.CheckPoints)-1], ",")
		_, err = db.Exec("UPDATE colis SET checkPointsDate = ? WHERE colisID = ?", newColisCheckpoints, colisID)
		if err != nil {
			return "", fmt.Errorf("erreur lors du retrait du checkpoint dans la base de données: %v", err)
		}

	} else if colis.Step <= colis.MaxStep {
		_, err = db.Exec("UPDATE colis SET step = step + 1 WHERE colisID = ?", colisID)
		if err != nil {
			return "", fmt.Errorf("erreur lors de l'incrémentation de la valeur de step dans la base de données: %v", err)
		}

		newColisCheckpoints := strings.Join(append(colis.CheckPoints, strconv.Itoa(int(getCurrentTimestamp()))), ",")
		_, err = db.Exec("UPDATE colis SET checkPointsDate = ? WHERE colisID = ?", newColisCheckpoints, colisID)
		if err != nil {
			return "", fmt.Errorf("erreur lors de l'ajout du checkpoint dans la base de données: %v", err)
		}
	}

	// Mettre à jour le transitID dans la base de données
	newTransitID := uuid.New().String()
	if colis.Step <= colis.MaxStep {
		_, err = db.Exec("UPDATE colis SET transitID = ? WHERE colisID = ?", newTransitID, colisID)
		if err != nil {
			return "", fmt.Errorf("erreur lors de la mise à jour du transitID dans la base de données: %v", err)
		}
	} else {
		_, err = db.Exec("UPDATE colis SET transitID = ? WHERE colisID = ?", "Transit fini", colisID)
		return "", fmt.Errorf("Transit du colis terminé")
	}

	// Renvoyer le nouveau transitID
	return newTransitID, nil
}

//==========
