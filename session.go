package xssdemo

import (
	"context"
	"fmt"
	"net/http"
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

const (
	masterClientID       = "ctf4b2022"
	masterClientPassword = "deadbeef"
)

func BasicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientID, clientSecret, ok := r.BasicAuth()
		if !ok || !(clientID == masterClientID && clientSecret == masterClientPassword) {
			logger.Printf("failed basicAuth: ok=%v, (%v, %v)\n", ok, clientID, clientSecret)
			w.Header().Add("WWW-Authenticate", `Basic realm="SECRET AREA"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func AuthUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Printf("AuthUser from %v", r.Header.Get("X-Forwarded-For"))

		// CookieからSession IDを取得する
		cookies := r.Cookies()
		sessionID, err := r.Cookie(sessionKey)
		// sessionIDが取得できなかった場合はそのまま処理を続ける
		if err != nil || sessionID == nil {
			logger.Printf("failed to get sessionID, cookies: %v\n", cookies)
			http.Redirect(w, r, "/users/login", http.StatusSeeOther)
			return
		}

		mu.Lock()
		defer mu.Unlock()
		user, ok := sessions[sessionID.Value]
		if !ok {
			logger.Printf("failed to find session: %v\n", sessionID.Value)
			http.Redirect(w, r, "/users/login", http.StatusSeeOther)
			return
		}

		// 無効なUserIDなのでDisableにする
		if user.ID <= 0 {
			logger.Printf("invalid user: %v\n", user)
			sessionID.MaxAge = -1
			http.SetCookie(w, sessionID)
			http.Redirect(w, r, "/users/login", http.StatusSeeOther)
			return
		}

		ctx := setToken(r.Context(), fmt.Sprintf("%d", user.ID))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
