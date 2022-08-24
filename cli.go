package xssdemo

import (
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
)

const addr = ":6060"

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
			fmt.Fprintf(os.Stderr, "failed to close DB: %s\n", err.Error())
		}
	}()

	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// パスで面倒が起き始めたら/postsや/usersに書き換える
	r.Mount("/", PostController{db}.Routes())
	r.Mount("/users", UserController{db}.Routes())

	fmt.Fprintf(os.Stderr, "Listen on %s\n", addr)
	http.ListenAndServe(addr, r)
	return nil
}
