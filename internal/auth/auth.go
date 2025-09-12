package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func CheckPasswordHash(password, hash string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return err
	}
	return nil
}

func GetBearerToken(headers http.Header) (string, error) {
	auth := headers.Get("Authorization")
	if auth == "" {
		return auth, errors.New("authorization header is not found")
	}
	token := strings.TrimSpace(auth[len("Bearer"):])
	if token != "" {
		return token, nil
	}
	return "", errors.New("token is empty")
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	claims := &jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
		Subject:   userID.String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}
	return ss, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (any, error) {
			return []byte(tokenSecret), nil
		},
	)
	if err != nil {
		return uuid.UUID{}, err
	}

	if !token.Valid {
		return uuid.UUID{}, errors.New("invalid token")
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return uuid.UUID{}, errors.New("unknown claims type")
	}

	parsedID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.UUID{}, err
	}

	return parsedID, nil
}
