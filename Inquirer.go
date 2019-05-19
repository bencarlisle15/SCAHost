package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"strconv"
	"time"
)

type UserData struct {
	User string `json:"user"`
	Hash string `json:"hash"`
	Salt string `json:"salt"`
	PublicKey string `json:"publickey"`
}

func CreateResults(rows *sql.Rows) []UserData {
	results := make([]UserData, 0)
	var currentUser UserData
	for rows.Next() {
		_ = rows.Scan(&currentUser.User, &currentUser.Hash, &currentUser.Salt, &currentUser.PublicKey)
		results = append(results, currentUser)
		break
	}
	return results
}

func AddUser(user string, hash, salt, publicKey []byte) bool {
	database, _ := sql.Open("sqlite3", "database.db")
	defer database.Close()
	rows, _ := database.Query("SELECT * FROM Users WHERE user=?", user)
	defer rows.Close()
	if rows.Next() {
		return false
	}
	statement, _ := database.Prepare("INSERT INTO Users VALUES (?, ?, ?, ?)")
	defer statement.Close()
	_, err := statement.Exec(user, hash, salt, publicKey)
	return err == nil
}

func GetUserData(user string) []UserData {
	database, _ := sql.Open("sqlite3", "database.db")
	defer database.Close()
	rows, _ := database.Query("SELECT * FROM Users WHERE user=?", user)
	defer rows.Close()
	return CreateResults(rows)
}

func AddSessionID(user string, sessionID string) bool {
	database, _ := sql.Open("sqlite3", "database.db")
	defer database.Close()
	statement, _ := database.Prepare("INSERT INTO Sessions VALUES (?, ?, ?)")
	defer statement.Close()
	_, err := statement.Exec(user, sessionID, GetEpoch())
	return err == nil
}

func SweepSessions() {
	database, _ := sql.Open("sqlite3", "database.db")
	defer database.Close()
	statement, _ := database.Prepare("DELETE FROM Sessions WHERE timestamp <= ?")
	defer statement.Close()
	rows,_ := statement.Exec(GetEpoch()-10)
	affected, _ := rows.RowsAffected()
	if affected > 0 {
		fmt.Println("Swept " + strconv.Itoa(int(affected)) + " sessions")
	}
}
func SweepMessages() {
	database, _ := sql.Open("sqlite3", "database.db")
	defer database.Close()
	statement, _ := database.Prepare("DELETE FROM Messages WHERE timestamp <= ?")
	defer statement.Close()
	rows, _ := statement.Exec(GetEpoch()-10)
	affected, _ := rows.RowsAffected()
	if affected > 0 {
		fmt.Println("Swept " + strconv.Itoa(int(affected)) + " messages")
	}
}

func GetSessions(user string) []string {
	database, _ := sql.Open("sqlite3", "database.db")
	defer database.Close()
	rows, _ := database.Query("SELECT sessionId FROM Sessions WHERE user=?", user)
	sessions := make([]string, 0)
	var sessionId string
	for rows.Next() {
		_ = rows.Scan(&sessionId)
		sessions = append(sessions, sessionId)
	}
	return sessions
}

func UpdateSession(sessionId string) {
	database, _ := sql.Open("sqlite3", "database.db")
	defer database.Close()
	statement, _ := database.Prepare("UPDATE Sessions SET timestamp=? WHERE sessionId=?")
	defer statement.Close()
	_,_ = statement.Exec(GetEpoch(), sessionId)
}

func GetNextMessage(user string) Sendable {
	database, _ := sql.Open("sqlite3", "database.db")
	defer database.Close()
	var sendable Sendable
	rows,_ := database.Query("SELECT * FROM Messages WHERE receiver=? LIMIT 1", user)
	if !rows.Next() {
		rows.Close()
		return sendable
	}
	//todo nonce
	_ = rows.Scan(&sendable.Receiver, &sendable.Sender, &sendable.Message, &sendable.IsFile, &sendable.Timestamp)
	rows.Close()
	database.Close()
	database, _ = sql.Open("sqlite3", "database.db")
	defer database.Close()
	statement, _ := database.Prepare("DELETE FROM Messages WHERE timestamp in (SELECT timestamp FROM Messages WHERE receiver=? AND sender=? AND message=? AND timestamp=?)")
	defer statement.Close()
	_,_ = statement.Exec(sendable.Receiver, sendable.Sender, sendable.Message, sendable.Timestamp)
	return sendable
}

func AddMessage(to string, from string, message string, isFile bool) {
	database, _ := sql.Open("sqlite3", "database.db")
	defer database.Close()
	statement, _ := database.Prepare("INSERT INTO Messages VALUES (?, ?, ?, ?, ?)")
	defer statement.Close()
	_, _ = statement.Exec(to, from, message, isFile, GetEpoch())
}

func GetEpoch() int64 {
	return time.Now().Unix()
}

func GetPublicKey(user string) string {
	userData := GetUserData(user)
	if len(userData) != 1 {
		return ""
	}
	return userData[0].PublicKey
}