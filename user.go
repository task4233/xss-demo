package xssdemo

import (
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type UserController struct {
	db *sqlx.DB
}

func (u UserController) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/signup", u.SignupForm)
	r.Post("/signup", u.Signup)
	r.Get("/login", u.LoginForm)
	r.Post("/login", u.Login)

	return r
}

var (
	//go:embed templates/login.html.tmpl
	loginForm string

	//go:embed templates/signup.html.tmpl
	signupForm string
)

type UserRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

func (u *UserRequest) Validate() error {
	if u.Name == "" {
		return errors.New("name must not be empty")
	}
	if u.Password == "" {
		return errors.New("password must not be empty")
	}
	return nil
}

type User struct {
	ID           int    `db:"id"`
	Name         string `db:"name"`
	PasswordHash string `db:"password_hash"`
}

func (u UserController) Login(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(os.Stderr, "Login called")

	session, err := r.Cookie(sessionKey)
	if err == nil && session != nil {
		if sess, ok := sessions[session.Value]; ok {
			setToken(r.Context(), fmt.Sprintf("%d", sess.ID))
			// http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	}

	// TODO:
	// request bodyからnameとpasswordを取得する
	userRequest := UserRequest{}
	err = json.NewDecoder(r.Body).Decode(&userRequest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed Decode JSON: %s\n", err.Error())
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// validation
	err = userRequest.Validate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed Validate: %s\n", err.Error())
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// nameでSELECTする
	user := User{}
	err = u.db.GetContext(r.Context(), &user, `SELECT * FROM users WHERE name = ?`, userRequest.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, fmt.Sprintf("user %s does not exist", userRequest.Name), http.StatusBadRequest)
			return
		}
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// passwordをハッシュ化する
	passwordHash := fmt.Sprintf("%x", sha256.Sum256([]byte(userRequest.Password)))

	// 比較して一致していたらSessionを設定する
	if passwordHash != user.PasswordHash {
		http.Error(w, "bad request", http.StatusUnauthorized)
		return
	}

	fmt.Fprintf(os.Stderr, "logged in: %v\n", user)

	sessionID := uuid.NewString()
	mu.Lock()
	sessions[sessionID] = user
	mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:  sessionKey,
		Value: sessionID,
		Path:  "/",
	})

	fmt.Fprintf(os.Stderr, "sessions: %v\n", sessions)

	setToken(r.Context(), fmt.Sprintf("%d", user.ID))
	// /にリダイレクトする
	// http.Redirect(w, r, "/", http.StatusSeeOther)
	// エラーが発生した場合は/loginにエラーメッセージのquery parameter付きでリダイレクトする
}

func (u UserController) LoginForm(w http.ResponseWriter, r *http.Request) {
	// TODO:
	// query parameterからエラー情報を受け取る
	// template/login.html.tmplを返却する
	errString := r.URL.Query().Get("error")
	renderedHTML, err := RenderTemplate(loginForm, errString)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed RenderTemplate: %s\n", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, renderedHTML)
}

func (u UserController) Signup(w http.ResponseWriter, r *http.Request) {
	// TODO:
	// request bodyからnameとpasswordを取得する
	userRequest := UserRequest{}
	err := json.NewDecoder(r.Body).Decode(&userRequest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed Decode JSON: %s\n", err.Error())
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// validation
	err = userRequest.Validate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed Validate: %s\n", err.Error())
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// passwordをハッシュ化する
	user := User{}
	user.Name = userRequest.Name
	user.PasswordHash = fmt.Sprintf("%x", sha256.Sum256([]byte(userRequest.Password)))

	res, err := u.db.NamedExecContext(r.Context(), `INSERT INTO users (name, password_hash) VALUES (:name, :password_hash)`, user)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed NamedExecContext: %s\n", err.Error())
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	fmt.Fprintf(os.Stderr, "result: %v\n", res)

	sessionID := uuid.NewString()
	mu.Lock()
	sessions[sessionID] = user
	mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:  sessionKey,
		Value: sessionID,
	})

	setToken(r.Context(), fmt.Sprintf("%d", user.ID))
	// /にリダイレクト
	// エラーが発生した場合は/signupにエラーメッセージのquery parameter付きでリダイレクト
}

func (u UserController) SignupForm(w http.ResponseWriter, r *http.Request) {
	// TODO:
	// query parameterからエラー情報を受け取る
	// template/signup.html.tmplを返却する
	errString := r.URL.Query().Get("error")
	renderedHTML, err := RenderTemplate(signupForm, errString)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed RenderTemplate: %s\n", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, renderedHTML)
}
