package server

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// --- Web auth (JWT for the bundled UI) ---

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	user := s.cfg.AuthenticateUser(req.Username, req.Password)
	if user == nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": user.Username,
		"exp":  time.Now().Add(30 * 24 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	})
	tokenStr, err := token.SignedString([]byte(s.cfg.App.JWTSecret))
	if err != nil {
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": tokenStr})
}

func (s *Server) jwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := ""
		if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			tokenStr = strings.TrimPrefix(auth, "Bearer ")
		} else if t := r.URL.Query().Get("token"); t != "" {
			tokenStr = t
		}
		if tokenStr == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return []byte(s.cfg.App.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- Subsonic auth ---
//
// Two flavors: classic `p=password` (sometimes hex-prefixed `enc:<hex>`),
// and the salted-token form `t=md5(password+salt)&s=<salt>`. We accept
// both. Username comes in `u`.

func (s *Server) checkSubsonicAuth(r *http.Request) (username string, ok bool) {
	q := r.URL.Query()
	username = q.Get("u")
	if username == "" {
		return "", false
	}
	expected := s.cfg.PasswordFor(username)
	if expected == "" {
		return "", false
	}

	if t, salt := q.Get("t"), q.Get("s"); t != "" && salt != "" {
		sum := md5.Sum([]byte(expected + salt))
		if hex.EncodeToString(sum[:]) == strings.ToLower(t) {
			return username, true
		}
		return "", false
	}

	if p := q.Get("p"); p != "" {
		if strings.HasPrefix(p, "enc:") {
			decoded, err := hex.DecodeString(strings.TrimPrefix(p, "enc:"))
			if err == nil && string(decoded) == expected {
				return username, true
			}
			return "", false
		}
		if p == expected {
			return username, true
		}
	}
	return "", false
}

func (s *Server) subsonicAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.checkSubsonicAuth(r); !ok {
			writeSubsonicError(w, r, ErrCodeWrongAuth, "Wrong username or password")
			return
		}
		next.ServeHTTP(w, r)
	})
}
