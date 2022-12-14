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
	ID        int    `db:"id"`
	UserID    int    `db:"user_id"`
	Title     string `db:"title" json:"title"`
	Body      string `db:"body" json:"body"`
	Available bool   `db:"available"`
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
	logger.Printf("executed SQL: INSERT INTO posts (user_id, title, body) VALUES (%d, %s, %s)\n", post.UserID, post.Title, post.Body)
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
	// qが空なら全件検索、そうでなければタイトルのキーワード検索
	title := r.URL.Query().Get("title")
	posts := []Post{}
	if len(title) == 0 {
		err = p.db.Select(&posts, `SELECT * FROM posts WHERE user_id = ? AND available = 1 ORDER BY id DESC`, userIDStr)
		logger.Printf("executed SQL: `SELECT * FROM posts WHERE user_id = %s AND available = 1 ORDER BY id DESC\n", userIDStr)
		if err != nil {
			logger.Printf("failed Select: %s\n", err.Error())
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	} else {
		var rows *sqlx.Rows
		rows, err = p.db.NamedQueryContext(r.Context(), `SELECT * FROM posts WHERE user_id = :user_id AND available = 1 AND title LIKE :title ORDER BY id DESC`, map[string]interface{}{
			"user_id": userIDStr,
			"title":   fmt.Sprintf("%%%s%%", title),
		})
		logger.Printf("executed SQL: SELECT * FROM posts WHERE user_id = %s AND available = 1 AND title LIKE %%%s%% ORDER BY id \n", userIDStr, title)
		if err != nil {
			logger.Printf("failed NamedQueryContext: %s", err.Error())
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer func() {
			if err = rows.Close(); err != nil {
				logger.Printf("failed rows.Close: %s", err.Error())
			}
		}()

		var post Post
		for rows.Next() {
			err = rows.StructScan(&post)
			if err != nil {
				logger.Printf("failed StructScan: %s\n", err.Error())
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			posts = append(posts, post)
		}
	}

	// /のHTMLを返す
	renderedHTML, err := RenderTemplate(listTemplates, struct {
		Title string
		Posts []Post
	}{
		Title: title,
		Posts: posts,
	})
	if err != nil {
		logger.Printf("failed RenderTemplate: %s\n", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, renderedHTML)
}

func (p PostController) Delete(w http.ResponseWriter, r *http.Request) {
	logger.Printf("DELETE / from %s", r.Header.Get("X-Forwarded-For"))

	id := chi.URLParam(r, "id")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		logger.Printf("failed Atoi: %s\n", id)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	userID, err := getToken(r.Context())
	if err != nil {
		logger.Printf("failed getToken: %s\n", err.Error())
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	userIDInt, err := strconv.Atoi(userID)
	if err != nil {
		logger.Printf("failed Atoi: %s\n", id)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// delete
	// userIDは他人の投稿を不正にUpdateされないようにするために付与する
	// どのような投稿をしたか後で見たいので、availableフラグを0にすることで削除された投稿も永続化する
	_, err = p.db.NamedExecContext(r.Context(), `UPDATE posts SET available=0 WHERE id=:id AND user_id=:user_id`, &Post{
		ID:     idInt,
		UserID: userIDInt,
	})
	logger.Printf("executed SQL: UPDATE posts SET available=0 WHERE id=%d AND user_id=%d\n", idInt, userIDInt)
	if err != nil {
		// 他人のpostを削除された時はエラーハンドリングがめんどくさいので、ログを出して対処したことにする
		logger.Printf("failed NamedExecContext of Delete: %s\n", err.Error())
		return
	}
}

func (p PostController) Routes() chi.Router {
	r := chi.NewRouter()

	r.Use(AuthUser)
	r.Get("/", p.List)
	r.Post("/", p.Create)
	r.Delete("/{id}", p.Delete)

	return r
}
