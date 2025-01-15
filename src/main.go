package main

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/caarlos0/env/v11"
	_ "github.com/lib/pq"
)

var (
	db *sql.DB

	//go:embed html
	res          embed.FS
	page         = "html/index.gohtml"
	databaseName = "notebook"
)

type Note struct {
	ID      int    `json:"id"`
	Content string `json:"content"`
}

type config struct {
	Database struct {
		Host     string `env:"HOST,required"`
		Password string `env:"PASSWORD,required"`
		Username string `env:"USERNAME,required"`
		Port     int    `env:"PORT,required"`
		Name     string `env:"NAME,required"`
	} `envPrefix:"DB_"`
}

func main() {
	cfg, err := env.ParseAs[config]()
	pgConnStr := fmt.Sprintf("dbname=%s user=%s password=%s host=%s port=%d", cfg.Database.Name, cfg.Database.Username, cfg.Database.Password, cfg.Database.Host, cfg.Database.Host)
	conn, err := sql.Open("postgres", pgConnStr)
	if err != nil {
		log.Fatalf("Error opening database connection: %v", err)
	}
	db = conn

	err = db.Ping()
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}
	fmt.Println("Connected to the PostgreSQL database")

	initialize()

	http.HandleFunc("/", indexPage)
	http.HandleFunc("GET /api/notes", getNotes)
	http.HandleFunc("POST /api/note", addNote)

	fmt.Println("Server is listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// func createDatabase() {
// 	query := fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname = '%s'", databaseName)
// 	result, err := db.Exec(query)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
//
// 	rowAffected, err := result.RowsAffected()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
//
// 	if rowAffected > 0 {
// 		fmt.Println("Database exists")
// 	} else {
// 		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", databaseName))
// 		if err != nil {
// 			log.Fatalf("Failed to create database: %v", err)
// 		}
// 		log.Printf("Database '%s' created successfully.\n", databaseName)
// 	}
// }

func indexPage(w http.ResponseWriter, r *http.Request) {
	tpl, err := template.ParseFS(res, page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	data := map[string]any{
		"notes": getNoteList(),
	}
	if err := tpl.Execute(w, data); err != nil {
		return
	}
}

func initialize() {
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS notes (
		id SERIAL PRIMARY KEY,
		content TEXT NOT NULL
	);
	`

	_, err := db.Exec(createTableQuery)
	if err != nil {
		log.Fatalf("Error creating table: %v", err)
	}
}

func getNoteList() []Note {
	rows, err := db.Query("SELECT id, content FROM notes")
	if err != nil {
		log.Fatalf("Error getting notes: %v", err)
		return nil
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var note Note
		err := rows.Scan(&note.ID, &note.Content)
		if err != nil {
			log.Fatalf("Error getting notes: %v", err)
			return nil
		}
		notes = append(notes, note)
	}
	return notes
}

func getNotes(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, content FROM notes")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var note Note
		err := rows.Scan(&note.ID, &note.Content)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		notes = append(notes, note)
	}

	err = json.NewEncoder(w).Encode(notes)
	if err != nil {
		log.Fatal("Error getting notes, while encoding")
	}
}

func addNote(w http.ResponseWriter, r *http.Request) {
	var note Note
	err := json.NewDecoder(r.Body).Decode(&note)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("INSERT INTO notes (content) VALUES ($1)", note.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Note created succcessfully")
}
