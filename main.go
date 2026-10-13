package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"
)

var db *sql.DB
var jwtSecret = []byte("super-secret-key-change-in-production")

func sendJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Println("JSON Encode Error:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type Book struct {
	ID          string  `json:"id"`
	Title       *string `json:"title,omitempty"`
	Pages       *int    `json:"pages,omitempty"`
	PublisherID *string `json:"PublisherID,omitempty"`
	GenreID     *string `json:"GenreID,omitempty"`
	AuthorID    *string `json:"AuthorID,omitempty"`
	Genre       *string `json:"genre,omitempty"`
	Publisher   *string `json:"publisher,omitempty"`
	Author      *string `json:"author,omitempty"`
}

type Publisher struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Genre struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Author struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func getBooks(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, title, CAST(pages AS INTEGER), publisher_id, genre_id, author_id FROM books")
	if err != nil {
		log.Println("Database Error:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	list := []Book{}
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Pages, &b.PublisherID, &b.GenreID, &b.AuthorID); err != nil {
			log.Println("Scan Error:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		list = append(list, b)
	}
	if err := rows.Err(); err != nil {
		log.Println("Row Iteration Error:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sendJSON(w, http.StatusOK, list)
}

func getBook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var b Book
	err := db.QueryRow("SELECT id, title, pages, publisher_id, genre_id, author_id FROM books WHERE id = $1", id).
		Scan(&b.ID, &b.Title, &b.Pages, &b.PublisherID, &b.GenreID, &b.AuthorID)

	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "Book not found"}`, http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sendJSON(w, http.StatusOK, b)
}

func borrowBook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var currentlyTaken bool
	err := db.QueryRow("SELECT status FROM status WHERE book_id = $1 ORDER BY taken DESC LIMIT 1", id).Scan(&currentlyTaken)

	if err != nil && err != sql.ErrNoRows {
		http.Error(w, `{"error": "Database error checking status"}`, http.StatusInternalServerError)
		return
	}

	if currentlyTaken {
		http.Error(w, `{"error": "Book is already checked out"}`, http.StatusConflict)
		return
	}

	_, err = db.Exec(`
		INSERT INTO status (book_id, status, taken, returned) 
		VALUES ($1, true, CURRENT_TIMESTAMP, NULL)
	`, id)

	if err != nil {
		http.Error(w, `{"error": "Failed to update status"}`, http.StatusInternalServerError)
		return
	}

	sendJSON(w, http.StatusOK, map[string]string{
		"message": "Book successfully borrowed!",
		"book_id": id,
	})
}

func returnBook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	result, err := db.Exec(`
		UPDATE status 
		SET status = false, returned = CURRENT_TIMESTAMP 
		WHERE book_id = $1 AND status = true
	`, id)

	if err != nil {
		http.Error(w, `{"error": "Failed to update status"}`, http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, `{"error": "Error checking update result"}`, http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, `{"error": "Book is not currently checked out"}`, http.StatusBadRequest)
		return
	}

	sendJSON(w, http.StatusOK, map[string]string{
		"message": "Book successfully returned!",
		"book_id": id,
	})
}

func createFullBook(w http.ResponseWriter, r *http.Request) {
	var b Book
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}

	err := db.QueryRow("INSERT INTO books (title, pages, publisher_id, genre_id, author_id) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		b.Title, b.Pages, b.PublisherID, b.GenreID, b.AuthorID).Scan(&b.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sendJSON(w, http.StatusCreated, b)
}

func updateBook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var b Book
	err := db.QueryRow("SELECT title, pages, publisher_id, genre_id, author_id FROM books WHERE id = $1", id).
		Scan(&b.Title, &b.Pages, &b.PublisherID, &b.GenreID, &b.AuthorID)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "Book not found"}`, http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	b.ID = id

	var updates map[string]string
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if title, ok := updates["title"]; ok {
		b.Title = &title
	}
	if pages, ok := updates["pages"]; ok {
		if n, err := strconv.Atoi(pages); err == nil {
			b.Pages = &n
		} else {
			http.Error(w, `{"error": "Invalid pages"}`, http.StatusBadRequest)
			return
		}
	}
	if pubID, ok := updates["PublisherID"]; ok {
		b.PublisherID = &pubID
	}
	if genID, ok := updates["GenreID"]; ok {
		b.GenreID = &genID
	}
	if authID, ok := updates["AuthorID"]; ok {
		b.AuthorID = &authID
	}

	_, err = db.Exec("UPDATE books SET title=$1, pages=$2, publisher_id=$3, genre_id=$4, author_id=$5 WHERE id=$6",
		b.Title, b.Pages, b.PublisherID, b.GenreID, b.AuthorID, b.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sendJSON(w, http.StatusOK, b)
}

func createBook(w http.ResponseWriter, r *http.Request) {
	var b Book
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}
	if b.Title != nil {
		s := strings.TrimSpace(*b.Title)
		b.Title = &s
	}
	if b.Title == nil || *b.Title == "" {
		http.Error(w, `{"error": "Title cannot be empty"}`, http.StatusBadRequest)
		return
	}

	if b.Pages == nil || *b.Pages < 1 {
		http.Error(w, `{"error": "Pages must be 1 or greater"}`, http.StatusBadRequest)
		return
	}

	if b.Author != nil {
		s := strings.TrimSpace(*b.Author)
		b.Author = &s
	}
	if b.Genre != nil {
		s := strings.TrimSpace(*b.Genre)
		b.Genre = &s
	}
	if b.Publisher != nil {
		s := strings.TrimSpace(*b.Publisher)
		b.Publisher = &s
	}

	hasAuthorID := b.AuthorID != nil && strings.TrimSpace(*b.AuthorID) != ""
	hasAuthorName := b.Author != nil && *b.Author != ""
	if !hasAuthorID && !hasAuthorName {
		http.Error(w, `{"error": "Valid Author ID or Name is required"}`, http.StatusBadRequest)
		return
	}

	hasGenreID := b.GenreID != nil && strings.TrimSpace(*b.GenreID) != ""
	hasGenreName := b.Genre != nil && *b.Genre != ""
	if !hasGenreID && !hasGenreName {
		http.Error(w, `{"error": "Valid Genre ID or Name is required"}`, http.StatusBadRequest)
		return
	}

	hasPubID := b.PublisherID != nil && strings.TrimSpace(*b.PublisherID) != ""
	hasPubName := b.Publisher != nil && *b.Publisher != ""
	if !hasPubID && !hasPubName {
		http.Error(w, `{"error": "Valid Publisher ID or Name is required"}`, http.StatusBadRequest)
		return
	}

	if !hasAuthorID && hasAuthorName {
		var newAuthorID string
		err := db.QueryRow("INSERT INTO authors (name) VALUES ($1) RETURNING id", *b.Author).Scan(&newAuthorID)
		if err != nil {
			http.Error(w, "Failed to create author: "+err.Error(), http.StatusInternalServerError)
			return
		}
		b.AuthorID = &newAuthorID
	}

	if !hasPubID && hasPubName {
		var newPubID string
		err := db.QueryRow("INSERT INTO publishers (name) VALUES ($1) RETURNING id", *b.Publisher).Scan(&newPubID)
		if err != nil {
			http.Error(w, "Failed to create publisher: "+err.Error(), http.StatusInternalServerError)
			return
		}
		b.PublisherID = &newPubID
	}

	if !hasGenreID && hasGenreName {
		var newGenreID string
		err := db.QueryRow("INSERT INTO genres (name) VALUES ($1) RETURNING id", *b.Genre).Scan(&newGenreID)
		if err != nil {
			http.Error(w, "Failed to create genre: "+err.Error(), http.StatusInternalServerError)
			return
		}
		b.GenreID = &newGenreID
	}

	err := db.QueryRow("INSERT INTO books (title, pages, publisher_id, genre_id, author_id) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		b.Title, b.Pages, b.PublisherID, b.GenreID, b.AuthorID).Scan(&b.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sendJSON(w, http.StatusCreated, b)
}

func getPublishers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name FROM publishers")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	list := []Publisher{}
	for rows.Next() {
		var p Publisher
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			http.Error(w, `{"error": "Internal"}`, http.StatusInternalServerError)
			return
		}
		list = append(list, p)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, `{"error": "Internal"}`, http.StatusInternalServerError)
		return
	}
	sendJSON(w, http.StatusOK, list)
}

func getPublisher(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var p Publisher
	err := db.QueryRow("SELECT id, name FROM publishers WHERE id = $1", id).Scan(&p.ID, &p.Name)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "Not found"}`, http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sendJSON(w, http.StatusOK, p)
}

func createPublisher(w http.ResponseWriter, r *http.Request) {
	var p Publisher
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}

	err := db.QueryRow("INSERT INTO publishers (name) VALUES ($1) RETURNING id", p.Name).Scan(&p.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sendJSON(w, http.StatusCreated, p)
}

func updatePublisher(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var p Publisher
	if err := db.QueryRow("SELECT name FROM publishers WHERE id = $1", id).Scan(&p.Name); err != nil {
		http.Error(w, `{"error": "Not found"}`, http.StatusNotFound)
		return
	}
	p.ID = id
	var updates map[string]string
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}
	if name, ok := updates["name"]; ok {
		p.Name = name
	}

	if _, err := db.Exec("UPDATE publishers SET name=$1 WHERE id=$2", p.Name, p.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sendJSON(w, http.StatusOK, p)
}

func getGenres(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name FROM genres")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	list := []Genre{}
	for rows.Next() {
		var g Genre
		rows.Scan(&g.ID, &g.Name)
		list = append(list, g)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sendJSON(w, http.StatusOK, list)
}

func getGenre(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var g Genre
	err := db.QueryRow("SELECT id, name FROM genres WHERE id = $1", id).Scan(&g.ID, &g.Name)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "Not found"}`, http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sendJSON(w, http.StatusOK, g)
}

func createGenre(w http.ResponseWriter, r *http.Request) {
	var g Genre
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}

	err := db.QueryRow("INSERT INTO genres (name) VALUES ($1) RETURNING id", g.Name).Scan(&g.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sendJSON(w, http.StatusCreated, g)
}

func updateGenre(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var g Genre
	if err := db.QueryRow("SELECT name FROM genres WHERE id = $1", id).Scan(&g.Name); err != nil {
		http.Error(w, `{"error": "Not found"}`, http.StatusNotFound)
		return
	}
	g.ID = id
	var updates map[string]string
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}
	if name, ok := updates["name"]; ok {
		g.Name = name
	}

	if _, err := db.Exec("UPDATE genres SET name=$1 WHERE id=$2", g.Name, g.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sendJSON(w, http.StatusOK, g)
}

func getAuthors(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name FROM authors")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	list := []Author{}
	for rows.Next() {
		var a Author
		if err := rows.Scan(&a.ID, &a.Name); err != nil {
			http.Error(w, `{"error": "Internal"}`, http.StatusInternalServerError)
			return
		}
		list = append(list, a)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, `{"error": "Internal"}`, http.StatusInternalServerError)
		return
	}
	sendJSON(w, http.StatusOK, list)
}

func getAuthor(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var a Author
	err := db.QueryRow("SELECT id, name FROM authors WHERE id = $1", id).Scan(&a.ID, &a.Name)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "Not found"}`, http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sendJSON(w, http.StatusOK, a)
}

func createAuthor(w http.ResponseWriter, r *http.Request) {
	var a Author
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}

	err := db.QueryRow("INSERT INTO authors (name) VALUES ($1) RETURNING id", a.Name).Scan(&a.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sendJSON(w, http.StatusCreated, a)
}

func updateAuthor(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var a Author
	if err := db.QueryRow("SELECT name FROM authors WHERE id = $1", id).Scan(&a.Name); err != nil {
		http.Error(w, `{"error": "Not found"}`, http.StatusNotFound)
		return
	}
	a.ID = id
	var updates map[string]string
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}
	if name, ok := updates["name"]; ok {
		a.Name = name
	}

	if _, err := db.Exec("UPDATE authors SET name=$1 WHERE id=$2", a.Name, a.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sendJSON(w, http.StatusOK, a)
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func JWTAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/users" || r.URL.Path == "/login" || r.URL.Path == "/token" {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, `{"error": "Missing or invalid Authorization header"}`, http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, http.ErrAbortHandler
			}
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, `{"error": "Invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func generateClientToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClientName string `json:"client_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.ClientName) == "" {
		req.ClientName = "guest"
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"client": req.ClientName,
		"exp":    time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		http.Error(w, `{"error": "Could not generate token"}`, http.StatusInternalServerError)
		return
	}

	sendJSON(w, http.StatusOK, map[string]string{
		"message": "Token generated successfully",
		"token":   tokenString,
	})
}

func createUser(w http.ResponseWriter, r *http.Request) {
	var u User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if u.Name == "" || u.Email == "" {
		http.Error(w, `{"error": "Name and Email are required"}`, http.StatusBadRequest)
		return
	}
	err := db.QueryRow("INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id", u.Name, u.Email).Scan(&u.ID)
	if err != nil {
		log.Println("Error inserting user:", err)
		http.Error(w, `{"error": "Failed to create user"}`, http.StatusInternalServerError)
		return
	}

	sendJSON(w, http.StatusCreated, u)
}

func loginUser(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}

	var id int
	var name string

	err := db.QueryRow("SELECT id, name FROM users WHERE email = $1", creds.Email).Scan(&id, &name)

	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "Invalid email or password"}`, http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Println("Login error:", err)
		http.Error(w, `{"error": "Internal Server Error"}`, http.StatusInternalServerError)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": id,
		"email":   creds.Email,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		http.Error(w, `{"error": "Could not generate token"}`, http.StatusInternalServerError)
		return
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Login successful",
		"token":   tokenString,
		"user": map[string]interface{}{
			"id":    id,
			"name":  name,
			"email": creds.Email,
		},
	})
}

func initDB() {
	var err error
	db, err = sql.Open("postgres", "postgresql://postgres:Justineatsdicks123%40atime@localhost:5432/postgres?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS publishers (id SERIAL PRIMARY KEY, name TEXT);
	CREATE TABLE IF NOT EXISTS genres (id SERIAL PRIMARY KEY, name TEXT);
	CREATE TABLE IF NOT EXISTS authors (id SERIAL PRIMARY KEY, name TEXT);
	CREATE TABLE IF NOT EXISTS books (
		id SERIAL PRIMARY KEY, 
		title TEXT, 
		pages TEXT, 
		publisher_id TEXT, 
		genre_id TEXT, 
		author_id TEXT
	);
	CREATE TABLE IF NOT EXISTS status (
		id serial4 NOT NULL,
		book_id int4 NOT NULL,
		status boolean not NULL,
		taken timestamp DEFAULT CURRENT_TIMESTAMP not NULL, 
		returned timestamp default NULL,
		CONSTRAINT status_pkey PRIMARY KEY (id)
	);`

	if _, err := db.Exec(schema); err != nil {
		log.Fatal(err)
	}
}

func main() {
	initDB()
	defer db.Close()

	http.HandleFunc("POST /users", createUser)
	http.HandleFunc("POST /login", loginUser)
	http.HandleFunc("POST /token", generateClientToken)

	http.HandleFunc("GET /books", getBooks)
	http.HandleFunc("GET /books/{id}", getBook)
	http.HandleFunc("POST /books/{id}/borrow", borrowBook)
	http.HandleFunc("POST /books/{id}/return", returnBook)
	http.HandleFunc("POST /books", createBook)
	http.HandleFunc("PATCH /books/{id}", updateBook)

	http.HandleFunc("GET /publishers", getPublishers)
	http.HandleFunc("GET /publishers/{id}", getPublisher)
	http.HandleFunc("POST /publishers", createPublisher)
	http.HandleFunc("PATCH /publishers/{id}", updatePublisher)

	http.HandleFunc("GET /genres", getGenres)
	http.HandleFunc("GET /genres/{id}", getGenre)
	http.HandleFunc("POST /genres", createGenre)
	http.HandleFunc("PATCH /genres/{id}", updateGenre)

	http.HandleFunc("GET /authors", getAuthors)
	http.HandleFunc("GET /authors/{id}", getAuthor)
	http.HandleFunc("POST /authors", createAuthor)
	http.HandleFunc("PATCH /authors/{id}", updateAuthor)

	log.Println("Running on :8080...")

	protectedMux := JWTAuthMiddleware(http.DefaultServeMux)
	log.Fatal(http.ListenAndServe(":8080", enableCORS(protectedMux)))
}
