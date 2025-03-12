package middleware

import (
	"context"
	"deployment-restarter/pkg/auth"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strings"
)

func JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		settings, _ := auth.GetSettings()
		var tokenStr string
		if cookie, err := r.Cookie("token"); err == nil {
			tokenStr = cookie.Value
		} else if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer") {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer")
		}
		if tokenStr == "" {
			http.Error(w, "Missing token", http.StatusUnauthorized)
			return
		}
		claims := &auth.Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return settings.JWTKey, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
		ctx := contextWithClaims(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type contextKey string

const claimsKey = contextKey("claims")

func contextWithClaims(ctx context.Context, claims *auth.Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

func GetClaims(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*auth.Claims)
	return claims, ok
}
