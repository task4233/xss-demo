package xssdemo

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gomodule/redigo/redis"
)

type contextKey string

const (
	sessionKey          = "SESSION_ID"
	contextKeyAuthToken = contextKey("auth-token")
	redisAddr           = "redis:6379"
)

var (
	pool *redis.Pool
)

func init() {
	pool = &redis.Pool{
		MaxActive:   10,
		MaxIdle:     5,
		IdleTimeout: time.Second * 60,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", redisAddr)
		},
	}
}

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

		userID, err := getUserIDBySessionID(sessionID.Value)
		if err != nil {
			logger.Printf("failed to find session: %v\n", sessionID.Value)
			http.Redirect(w, r, "/users/login", http.StatusSeeOther)
			return
		}

		// 無効なUserIDなのでDisableにする
		if userID <= 0 {
			logger.Printf("invalid user: %v\n", userID)
			DisableCookie(&w, sessionID)
			http.Redirect(w, r, "/users/login", http.StatusSeeOther)
			return
		}

		ctx := setToken(r.Context(), fmt.Sprintf("%d", userID))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func DisableCookie(w *http.ResponseWriter, c *http.Cookie) {
	c.Value = ""
	c.MaxAge = -1
	c.Path = "/"
	c.Domain = ""
	c.Secure = false
	c.HttpOnly = false

	http.SetCookie(*w, c)
}

func storeSession(key string, id int) error {
	conn := pool.Get()
	defer conn.Close()

	// これ以降に処理がないので、err checkをせずにそのまま返す
	res, err := conn.Do("HSET", key, "id", id)
	if err != nil {
		return err
	}

	logger.Printf("result of HSET: %v\n", res)
	return nil
}

func getUserIDBySessionID(key string) (int, error) {
	conn := pool.Get()
	defer conn.Close()

	res, err := redis.Values(conn.Do("HGETALL", key))
	if err != nil {
		logger.Printf("failed redis.Values: %s, %v", key, res)
		return -1, err
	}

	u := &User{}
	err = redis.ScanStruct(res, u)
	if err != nil {
		logger.Printf("failed ScanStruct: %v, %s", res, err.Error())
		return -1, err
	}

	return u.ID, nil
}
