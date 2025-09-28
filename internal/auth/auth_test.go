package auth_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/prchop/chirpysrv/internal/auth"
)

const (
	ID1   = "c5b48091-f192-479b-8b18-b5db547c1eff"
	ID2   = "c49740d1-f27e-4a89-90cb-8472b682585c"
	ID3   = "e6aa2ddb-4c6f-449d-a366-b96c39311ed5"
	HMAC1 = "my_secret1"
	HMAC2 = "my_secret2"
	HMAC3 = "my_secret3"
)

var (
	listToken          []string
	ErrorExpiredToken  = fmt.Errorf("%w: %w", jwt.ErrTokenInvalidClaims, jwt.ErrTokenExpired)
	ErrorInvalidToken  = fmt.Errorf("%w: token contains an invalid number of segments", jwt.ErrTokenMalformed)
	ErrorInvalidSecret = fmt.Errorf("%w: %w", jwt.ErrTokenSignatureInvalid, jwt.ErrSignatureInvalid)
)

func TestCreateToken(t *testing.T) {
	parsedID1, _ := uuid.Parse(ID1)
	parsedID2, _ := uuid.Parse(ID2)
	parsedID3, _ := uuid.Parse(ID3)

	tests := []struct {
		id     uuid.UUID
		secret string
		expire time.Duration
		want   int
	}{
		{id: parsedID1, secret: HMAC1, expire: time.Second, want: 3},
		{id: parsedID2, secret: HMAC2, expire: time.Second * 2, want: 3},
		{id: parsedID3, secret: HMAC3, expire: time.Second * 3, want: 3},
	}

	for _, tt := range tests {
		testname := fmt.Sprintf("%v,%s", tt.id, tt.secret)
		t.Run(testname, func(t *testing.T) {
			got, err := auth.MakeJWT(tt.id, tt.secret, tt.expire)
			if err != nil {
				t.Fatal(err)
			}

			listToken = append(listToken, got)

			parts := strings.Split(got, ".")
			if len(parts) != tt.want {
				t.Fatalf(`got token: %d, want: %v`, len(got), tt.want)
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	token1 := listToken[0]
	token2 := listToken[1]
	token3 := listToken[2]

	parsedID1, _ := uuid.Parse(ID1)
	parsedID2, _ := uuid.Parse(ID2)
	parsedID3, _ := uuid.Parse(ID3)

	tests := []struct {
		token  string
		secret string
		delay  time.Duration
		want   uuid.UUID
	}{
		{token: token1, secret: HMAC1, delay: time.Second, want: parsedID1},
		{token: token2, secret: HMAC2, delay: time.Second, want: parsedID2},
		{token: token3, secret: HMAC3, delay: time.Second, want: parsedID3},
	}

	for _, tt := range tests {
		t.Run(tt.secret, func(t *testing.T) {
			got, err := auth.ValidateJWT(tt.token, tt.secret)
			if err != nil {
				t.Fatal(err)
			}

			time.Sleep(tt.delay)

			if got != tt.want {
				t.Fatalf(`got: %s, want: %v`, got, tt.want)
			}
		})
	}
}

func TestCheckValidateError(t *testing.T) {
	token1 := listToken[0]
	token2 := listToken[1]

	tests := []struct {
		name    string
		token   string
		secret  string
		delay   time.Duration
		wantErr error
	}{
		{name: "expired token", token: token1, secret: HMAC1, delay: time.Second * 2, wantErr: ErrorExpiredToken},
		{name: "invalid token", token: "testrejecttoken", secret: HMAC2, delay: time.Second, wantErr: ErrorInvalidToken},
		{name: "invalid secret", token: token2, secret: "testrejectsecret", delay: time.Second, wantErr: ErrorInvalidSecret},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := auth.ValidateJWT(tt.token, tt.secret)
			if err.Error() != tt.wantErr.Error() {
				t.Fatalf(`got: %q, want: %q`, err.Error(), tt.wantErr.Error())
			}
		})
	}
}

func TestGetBearerToken(t *testing.T) {
	tests := []struct {
		name    string
		headers http.Header
		want    string
		wantErr bool
	}{
		{"valid bearer token 1", http.Header{"Authorization": []string{"Bearer token12345"}}, "token12345", false},
		{"valid bearer token 2", http.Header{"Authorization": []string{"Bearer token54321"}}, "token54321", false},
		{"empty token", http.Header{"Authorization": []string{"Bearer "}}, "", true},
		{"authorization header not found", http.Header{"Authorization": []string{""}}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := auth.GetBearerToken(tt.headers)
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tt.want {
					t.Fatalf("got: %v, want: %v", got, tt.want)
				}
			} else {
				if err == nil {
					t.Fatalf("want error but got none")
				}
			}
		})
	}
}

func TestMakeRefreshToken(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"valid refresh token 1"},
		{"valid refresh token 2"},
		{"valid refresh token 3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := auth.MakeRefreshToken()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != 64 {
				t.Fatalf("expected length 32, got: %d", len(got))
			}
		})
	}
}
