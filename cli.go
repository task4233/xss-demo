package xssdemo

import (
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
)

var addr = fmt.Sprintf(":%s", os.Getenv("PORT"))

type Server struct{}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Run() error {
	db, err := NewDB()
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Printf("failed to close DB: %s\n", err.Error())
		}
	}()

	r := chi.NewRouter()
	r.Use(BasicAuth)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// パスで面倒が起き始めたら/postsや/usersに書き換える
	r.Mount("/", PostController{db}.Routes())
	r.Mount("/users", UserController{db}.Routes())

	logger.Printf("Listen on %s\n", addr)
	http.ListenAndServe(addr, r)

	return nil
}
