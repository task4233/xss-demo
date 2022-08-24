package xssdemo

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
)

type contextKey string

const (
	sessionKey          = "SESSION_ID"
	contextKeyAuthToken = contextKey("auth-token")
)

var (
	// map[UUID]User
	sessions map[string]User = make(map[string]User)
	mu       *sync.Mutex     = &sync.Mutex{}
)

func setToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, contextKeyAuthToken, token)
}

func getToken(ctx context.Context) (string, error) {
	if val, ok := ctx.Value(contextKeyAuthToken).(string); ok {
		return val, nil
	}
	return "", fmt.Errorf("failed conversion for session token: %v", ctx.Value(contextKeyAuthToken))
}

func AuthUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(os.Stderr, "AuthUser called")

		// CookieからSession IDを取得する
		cookies := r.Cookies()
		sessionID, err := r.Cookie(sessionKey)
		// sessionIDが取得できなかった場合はそのまま処理を続ける
		if err != nil || sessionID == nil {
			fmt.Fprintf(os.Stderr, "failed to get sessionID, cookies: %v\n", cookies)
			http.Redirect(w, r, "/users/login", http.StatusSeeOther)
			return
		}
		fmt.Fprintf(os.Stderr, "sessions: %v\n", sessions)
		fmt.Fprintf(os.Stderr, "cookies: %v\n", cookies)

		mu.Lock()
		defer mu.Unlock()
		user, ok := sessions[sessionID.Value]
		if !ok {
			fmt.Fprintf(os.Stderr, "failed to find session: %v\n", sessionID.Value)
			http.Redirect(w, r, "/users/login", http.StatusSeeOther)
			return
		}

		ctx := setToken(r.Context(), fmt.Sprintf("%d", user.ID))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
