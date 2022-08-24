package xssdemo

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

type PostController struct {
	db *sqlx.DB
}

var (
	//go:embed templates/post.html.tmpl
	postTemplate string
)

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
		fmt.Fprintf(os.Stderr, "failed getToken: %s\n", err.Error())
		http.Redirect(w, r, "/users/login", http.StatusSeeOther)
		return
	}
	post.UserID, err = strconv.Atoi(userIDStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid userID: %v\n", r.Context().Value(contextKeyAuthToken))
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err = post.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "failed Validate: %s", err.Error())
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// DBにinsert
	_, err = p.db.NamedExecContext(r.Context(), `INSERT INTO posts (user_id, title, body) VALUES (:user_id, :title, :body)`, post)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed insertion: %s", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	// /にGETでリダイレクト
	// これはフロントに任せる
}

func (p PostController) List(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(os.Stderr, "List called")

	userIDStr, err := getToken(r.Context())
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed getToken: %s\n", err.Error())
		http.Redirect(w, r, "/users/login", http.StatusSeeOther)
		return
	}

	// DBからSELECT
	posts := []Post{}
	err = p.db.Select(&posts, `SELECT * FROM posts WHERE user_id = ? ORDER BY id DESC`, userIDStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed Select: %s\n", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// /のHTMLを返す
	renderedHTML, err := RenderTemplate(postTemplate, posts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed RenderTemplate: %s\n", err.Error())
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
