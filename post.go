package xssdemo

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

type PostController struct {
	db *sqlx.DB
}

var listTemplates = []string{"templates/layout.html", "templates/post.html"}

type Post struct {
	ID     int    `db:"id"`
	UserID int    `db:"user_id"`
	Title  string `db:"title" json:"title"`
	Body   string `db:"body" json:"body"`
}

func (p *Post) Validate() error {
	if p.Title == "" {
		return errors.New("title must not be empty")
	}
	if p.Body == "" {
		return errors.New("body must not be empty")
	}
	return nil
}

func (p PostController) Create(w http.ResponseWriter, r *http.Request) {
	logger.Printf("POST / from %s", r.Header.Get("X-Forwarded-For"))

	// request bodyからtitle, bodyを抜き出す
	post := Post{}
	err := json.NewDecoder(r.Body).Decode(&post)
	if err != nil {
		fmt.Fprintf(w, "failed Decode JSON: %s", err.Error())
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// validation
	userIDStr, err := getToken(r.Context())
	if err != nil {
		logger.Printf("failed getToken: %s\n", err.Error())
		http.Redirect(w, r, "/users/login", http.StatusSeeOther)
		return
	}
	post.UserID, err = strconv.Atoi(userIDStr)
	if err != nil {
		logger.Printf("invalid userID: %v\n", r.Context().Value(contextKeyAuthToken))
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err = post.Validate(); err != nil {
		logger.Printf("failed Validate: %s", err.Error())
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	logger.Printf("new post: %v\n", post)

	// DBにinsert
	_, err = p.db.NamedExecContext(r.Context(), `INSERT INTO posts (user_id, title, body) VALUES (:user_id, :title, :body)`, post)
	if err != nil {
		logger.Printf("failed insertion: %s", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (p PostController) List(w http.ResponseWriter, r *http.Request) {
	logger.Printf("GET / from %s", r.Header.Get("X-Forwarded-For"))

	userIDStr, err := getToken(r.Context())
	if err != nil {
		logger.Printf("failed getToken: %s\n", err.Error())
		http.Redirect(w, r, "/users/login", http.StatusSeeOther)
		return
	}

	// DBからSELECT
	posts := []Post{}
	err = p.db.Select(&posts, `SELECT * FROM posts WHERE user_id = ? ORDER BY id DESC`, userIDStr)
	if err != nil {
		logger.Printf("failed Select: %s\n", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// /のHTMLを返す
	renderedHTML, err := RenderTemplate(listTemplates, posts)
	if err != nil {
		logger.Printf("failed RenderTemplate: %s\n", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, renderedHTML)
}

func (p PostController) Routes() chi.Router {
	r := chi.NewRouter()

	r.Use(AuthUser)
	r.Get("/", p.List)
	r.Post("/", p.Create)

	return r
}
