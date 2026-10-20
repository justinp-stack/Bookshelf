package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB
var jwtSecret []byte

func init() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}
	jwtSecret = []byte(secret)
}

func sendJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Println("JSON Encode Error:", err)
	}
}

func sendError(w http.ResponseWriter, status int, message string) {
	sendJSON(w, status, map[string]string{"error": message})
}

type Book struct {
	ID          string  `json:"id"`
	Title       *string `json:"title,omitempty"`
	Pages       *int    `json:"pages,omitempty"`
	PublisherID *int    `json:"publisher_id,omitempty"`
	GenreID     *int    `json:"genre_id,omitempty"`
	AuthorID    *int    `json:"author_id,omitempty"`
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
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password,omitempty"` // Used for incoming JSON only
}

func getBooks(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, title, pages, publisher_id, genre_id, author_id FROM books")
	if err != nil {
		log.Println("Database Error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	defer rows.Close()

	list := []Book{}
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Pages, &b.PublisherID, &b.GenreID, &b.AuthorID); err != nil {
			log.Println("Scan Error:", err)
			sendError(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		list = append(list, b)
	}
	if err := rows.Err(); err != nil {
		log.Println("Row Iteration Error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
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
		sendError(w, http.StatusNotFound, "Book not found")
		return
	} else if err != nil {
		log.Println("Database Error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	sendJSON(w, http.StatusOK, b)
}

func borrowBook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var existingBookID string
	err := db.QueryRow("SELECT id FROM books WHERE id = $1", id).Scan(&existingBookID)
	if err == sql.ErrNoRows {
		sendError(w, http.StatusNotFound, "Book not found")
		return
	} else if err != nil {
		log.Println("Database Error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	var currentlyTaken bool
	err = db.QueryRow("SELECT status FROM status WHERE book_id = $1 ORDER BY taken DESC LIMIT 1", id).Scan(&currentlyTaken)

	if err != nil && err != sql.ErrNoRows {
		log.Println("Database Error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	if currentlyTaken {
		sendError(w, http.StatusConflict, "Book is already checked out")
		return
	}

	_, err = db.Exec(`
		INSERT INTO status (book_id, status, taken, returned) 
		VALUES ($1, true, CURRENT_TIMESTAMP, NULL)
	`, id)

	if err != nil {
		log.Println("Database Error:", err)
		sendError(w, http.StatusInternalServerError, "Failed to update status")
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
		log.Println("Database Error:", err)
		sendError(w, http.StatusInternalServerError, "Failed to update status")
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Println("Database Error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	if rowsAffected == 0 {
		sendError(w, http.StatusBadRequest, "Book is not currently checked out")
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
		sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	err := db.QueryRow("INSERT INTO books (title, pages, publisher_id, genre_id, author_id) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		b.Title, b.Pages, b.PublisherID, b.GenreID, b.AuthorID).Scan(&b.ID)
	if err != nil {
		log.Println("Database Error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
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
		sendError(w, http.StatusNotFound, "Book not found")
		return
	} else if err != nil {
		log.Println("Database Error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	b.ID = id

	var updates struct {
		Title       *string `json:"title"`
		Pages       *int    `json:"pages"`
		PublisherID *int    `json:"publisher_id"`
		GenreID     *int    `json:"genre_id"`
		AuthorID    *int    `json:"author_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if updates.Title != nil {
		b.Title = updates.Title
	}
	if updates.Pages != nil {
		if *updates.Pages < 1 {
			sendError(w, http.StatusBadRequest, "Invalid pages")
			return
		}
		b.Pages = updates.Pages
	}
	if updates.PublisherID != nil {
		b.PublisherID = updates.PublisherID
	}
	if updates.GenreID != nil {
		b.GenreID = updates.GenreID
	}
	if updates.AuthorID != nil {
		b.AuthorID = updates.AuthorID
	}

	_, err = db.Exec("UPDATE books SET title=$1, pages=$2, publisher_id=$3, genre_id=$4, author_id=$5 WHERE id=$6",
		b.Title, b.Pages, b.PublisherID, b.GenreID, b.AuthorID, b.ID)
	if err != nil {
		log.Println("Database Error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	sendJSON(w, http.StatusOK, b)
}

func createBook(w http.ResponseWriter, r *http.Request) {
	var b Book
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if b.Title != nil {
		s := strings.TrimSpace(*b.Title)
		b.Title = &s
	}
	if b.Title == nil || *b.Title == "" {
		sendError(w, http.StatusBadRequest, "Title cannot be empty")
		return
	}

	if b.Pages == nil || *b.Pages < 1 {
		sendError(w, http.StatusBadRequest, "Pages must be 1 or greater")
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

	hasAuthorID := b.AuthorID != nil
	hasAuthorName := b.Author != nil && *b.Author != ""
	if !hasAuthorID && !hasAuthorName {
		sendError(w, http.StatusBadRequest, "Valid Author ID or Name is required")
		return
	}

	hasGenreID := b.GenreID != nil
	hasGenreName := b.Genre != nil && *b.Genre != ""
	if !hasGenreID && !hasGenreName {
		sendError(w, http.StatusBadRequest, "Valid Genre ID or Name is required")
		return
	}

	hasPubID := b.PublisherID != nil
	hasPubName := b.Publisher != nil && *b.Publisher != ""
	if !hasPubID && !hasPubName {
		sendError(w, http.StatusBadRequest, "Valid Publisher ID or Name is required")
		return
	}

	if !hasAuthorID && hasAuthorName {
		var authorID int
		err := db.QueryRow("SELECT id FROM authors WHERE name = $1", *b.Author).Scan(&authorID)
		if err == sql.ErrNoRows {
			err = db.QueryRow("INSERT INTO authors (name) VALUES ($1) RETURNING id", *b.Author).Scan(&authorID)
			if err != nil {
				log.Println("Database Error:", err)
				sendError(w, http.StatusInternalServerError, "Failed to create author")
				return
			}
		} else if err != nil {
			log.Println("Database Error:", err)
			sendError(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		b.AuthorID = &authorID
	}

	if !hasPubID && hasPubName {
		var pubID int
		err := db.QueryRow("SELECT id FROM publishers WHERE name = $1", *b.Publisher).Scan(&pubID)
		if err == sql.ErrNoRows {
			err = db.QueryRow("INSERT INTO publishers (name) VALUES ($1) RETURNING id", *b.Publisher).Scan(&pubID)
			if err != nil {
				log.Println("Database Error:", err)
				sendError(w, http.StatusInternalServerError, "Failed to create publisher")
				return
			}
		} else if err != nil {
			log.Println("Database Error:", err)
			sendError(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		b.PublisherID = &pubID
	}

	if !hasGenreID && hasGenreName {
		var genreID int
		err := db.QueryRow("SELECT id FROM genres WHERE name = $1", *b.Genre).Scan(&genreID)
		if err == sql.ErrNoRows {
			err = db.QueryRow("INSERT INTO genres (name) VALUES ($1) RETURNING id", *b.Genre).Scan(&genreID)
			if err != nil {
				log.Println("Database Error:", err)
				sendError(w, http.StatusInternalServerError, "Failed to create genre")
				return
			}
		} else if err != nil {
			log.Println("Database Error:", err)
			sendError(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		b.GenreID = &genreID
	}

	err := db.QueryRow("INSERT INTO books (title, pages, publisher_id, genre_id, author_id) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		b.Title, b.Pages, b.PublisherID, b.GenreID, b.AuthorID).Scan(&b.ID)
	if err != nil {
		log.Println("Database Error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	sendJSON(w, http.StatusCreated, b)
}

func getPublishers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name FROM publishers")
	if err != nil {
		log.Println("Database Error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	defer rows.Close()

	list := []Publisher{}
	for rows.Next() {
		var p Publisher
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			log.Println("Database Error:", err)
			sendError(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		list = append(list, p)
	}
	if err := rows.Err(); err != nil {
		log.Println("Database Error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	sendJSON(w, http.StatusOK, list)
}

func getPublisher(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var p Publisher
	err := db.QueryRow("SELECT id, name FROM publishers WHERE id = $1", id).Scan(&p.ID, &p.Name)
	if err == sql.ErrNoRows {
		sendError(w, http.StatusNotFound, "Not found")
		return
	} else if err != nil {
		log.Println("Database Error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	sendJSON(w, http.StatusOK, p)
}

func createPublisher(w http.ResponseWriter, r *http.Request) {
	var p Publisher
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	err := db.QueryRow("INSERT INTO publishers (name) VALUES ($1) RETURNING id", p.Name).Scan(&p.ID)
	if err != nil {
		log.Println("Database Error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	sendJSON(w, http.StatusCreated, p)
}

func updatePublisher(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var p Publisher
	if err := db.QueryRow("SELECT name FROM publishers WHERE id = $1", id).Scan(&p.Name); err != nil {
		sendError(w, http.StatusNotFound, "Not found")
		return
	}
	p.ID = id
	var updates map[string]string
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if name, ok := updates["name"]; ok {
		p.Name = name
	}

	if _, err := db.Exec("UPDATE publishers SET name=$1 WHERE id=$2", p.Name, p.ID); err != nil {
		log.Println("Database Error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	sendJSON(w, http.StatusOK, p)
}

func getGenres(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name FROM genres")
	if err != nil {
		log.Println("Database Error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	defer rows.Close()
	list := []Genre{}
	for rows.Next() {
		var g Genre
		if err := rows.Scan(&g.ID, &g.Name); err != nil {
			log.Println("Scan Error:", err)
			sendError(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		list = append(list, g)
	}
	if err := rows.Err(); err != nil {
		log.Println("Database Error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	sendJSON(w, http.StatusOK, list)
}

func getGenre(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var g Genre
	err := db.QueryRow("SELECT id, name FROM genres WHERE id = $1", id).Scan(&g.ID, &g.Name)
	if err == sql.ErrNoRows {
		sendError(w, http.StatusNotFound, "Not found")
		return
	} else if err != nil {
		log.Println("Database Error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	sendJSON(w, http.StatusOK, g)
}

func createGenre(w http.ResponseWriter, r *http.Request) {
	var g Genre
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	err := db.QueryRow("INSERT INTO genres (name) VALUES ($1) RETURNING id", g.Name).Scan(&g.ID)
	if err != nil {
		log.Println("Database Error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	sendJSON(w, http.StatusCreated, g)
}

func updateGenre(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var g Genre
	if err := db.QueryRow("SELECT name FROM genres WHERE id = $1", id).Scan(&g.Name); err != nil {
		sendError(w, http.StatusNotFound, "Not found")
		return
	}
	g.ID = id
	var updates map[string]string
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if name, ok := updates["name"]; ok {
		g.Name = name
	}

	if _, err := db.Exec("UPDATE genres SET name=$1 WHERE id=$2", g.Name, g.ID); err != nil {
		log.Println("Database Error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	sendJSON(w, http.StatusOK, g)
}

func getAuthors(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name FROM authors")
	if err != nil {
		log.Println("Database Error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	defer rows.Close()
	list := []Author{}
	for rows.Next() {
		var a Author
		if err := rows.Scan(&a.ID, &a.Name); err != nil {
			log.Println("Scan Error:", err)
			sendError(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		list = append(list, a)
	}
	if err := rows.Err(); err != nil {
		log.Println("Database Error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	sendJSON(w, http.StatusOK, list)
}

func getAuthor(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var a Author
	err := db.QueryRow("SELECT id, name FROM authors WHERE id = $1", id).Scan(&a.ID, &a.Name)
	if err == sql.ErrNoRows {
		sendError(w, http.StatusNotFound, "Not found")
		return
	} else if err != nil {
		log.Println("Database Error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	sendJSON(w, http.StatusOK, a)
}

func createAuthor(w http.ResponseWriter, r *http.Request) {
	var a Author
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	err := db.QueryRow("INSERT INTO authors (name) VALUES ($1) RETURNING id", a.Name).Scan(&a.ID)
	if err != nil {
		log.Println("Database Error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	sendJSON(w, http.StatusCreated, a)
}

func updateAuthor(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var a Author
	if err := db.QueryRow("SELECT name FROM authors WHERE id = $1", id).Scan(&a.Name); err != nil {
		sendError(w, http.StatusNotFound, "Not found")
		return
	}
	a.ID = id
	var updates map[string]string
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if name, ok := updates["name"]; ok {
		a.Name = name
	}

	if _, err := db.Exec("UPDATE authors SET name=$1 WHERE id=$2", a.Name, a.ID); err != nil {
		log.Println("Database Error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	sendJSON(w, http.StatusOK, a)
}

func deleteBook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	result, err := db.Exec("DELETE FROM books WHERE id = $1", id)
	if err != nil {
		log.Println("Database Delete Error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Println("Database Error:", err)
		sendError(w, http.StatusInternalServerError, "Error checking delete result")
		return
	}

	if rowsAffected == 0 {
		sendError(w, http.StatusNotFound, "Book not found")
		return
	}

	sendJSON(w, http.StatusOK, map[string]string{
		"message": "Book successfully deleted",
		"id":      id,
	})
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
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
		path := r.URL.Path
		if path == "/users" || path == "/login" || strings.HasPrefix(path, "/books") || strings.HasPrefix(path, "/genres") || strings.HasPrefix(path, "/publishers") || strings.HasPrefix(path, "/authors") {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			sendError(w, http.StatusUnauthorized, "Missing or invalid Authorization header")
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
			sendError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func createUser(w http.ResponseWriter, r *http.Request) {
	var u User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if u.Name == "" || u.Email == "" || u.Password == "" {
		sendError(w, http.StatusBadRequest, "Name, Email, and Password are required")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("Bcrypt Error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	err = db.QueryRow("INSERT INTO users (name, email, password_hash) VALUES ($1, $2, $3) RETURNING id", u.Name, u.Email, string(hash)).Scan(&u.ID)
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			sendError(w, http.StatusConflict, "Email already exists")
			return
		}
		log.Println("Error inserting user:", err)
		sendError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	u.Password = "" // Clear password before sending response
	sendJSON(w, http.StatusCreated, u)
}

func loginUser(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	var id int
	var name string
	var hash string

	err := db.QueryRow("SELECT id, name, password_hash FROM users WHERE email = $1", creds.Email).Scan(&id, &name, &hash)
	if err == sql.ErrNoRows {
		sendError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	} else if err != nil {
		log.Println("Login error:", err)
		sendError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(creds.Password)); err != nil {
		sendError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": id,
		"email":   creds.Email,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		log.Println("JWT Error:", err)
		sendError(w, http.StatusInternalServerError, "Could not generate token")
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

	dbUser := os.Getenv("DBUSER")
	if dbUser == "" {
		log.Fatal("DBUSER environment variable is required")
	}

	dbPass := os.Getenv("DBPASS")
	if dbPass == "" {
		log.Fatal("DBPASS environment variable is required")
	}

	dsn := fmt.Sprintf("host=localhost port=5432 user=%s password=%s dbname=postgres sslmode=disable", dbUser, dbPass)

	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY, 
		name TEXT NOT NULL, 
		email TEXT UNIQUE NOT NULL, 
		password_hash TEXT
	);
	CREATE TABLE IF NOT EXISTS publishers (id SERIAL PRIMARY KEY, name TEXT UNIQUE);
	CREATE TABLE IF NOT EXISTS genres (id SERIAL PRIMARY KEY, name TEXT UNIQUE);
	CREATE TABLE IF NOT EXISTS authors (id SERIAL PRIMARY KEY, name TEXT UNIQUE);
	CREATE TABLE IF NOT EXISTS books (
		id SERIAL PRIMARY KEY, 
		title TEXT, 
		pages INTEGER, 
		publisher_id INTEGER REFERENCES publishers(id), 
		genre_id INTEGER REFERENCES genres(id), 
		author_id INTEGER REFERENCES authors(id)
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

	http.HandleFunc("GET /books", getBooks)
	http.HandleFunc("GET /books/{id}", getBook)
	http.HandleFunc("POST /books/{id}/borrow", borrowBook)
	http.HandleFunc("POST /books/{id}/return", returnBook)
	http.HandleFunc("POST /books", createBook)
	http.HandleFunc("POST /books/full", createFullBook) // Kept and routed
	http.HandleFunc("PATCH /books/{id}", updateBook)
	http.HandleFunc("DELETE /books/{id}", deleteBook)

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
