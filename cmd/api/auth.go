package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"webapp/pkg/data"

	"github.com/golang-jwt/jwt/v4"
)

const jwtTokenExpiry = time.Minute * 15
const refreshTokenExpiry = time.Hour * 24

type TokenPairs struct {
	Token        string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Claims struct {
	UserName string `json:"name"`
	jwt.RegisteredClaims
}

func (app *application) getTokenFromHeaderAndVerify(w http.ResponseWriter, r *http.Request) (string, *Claims, error) {
	// add a header
	w.Header().Add("Vary", "Authorization")

	// get the authorization header
	authHeader := r.Header.Get("Authorization")

	// sanity check
	if authHeader == "" {
		return "", nil, errors.New("no auth header")
	}

	// split header on spaces
	headerParts := strings.Split(authHeader, " ")
	if len(headerParts) != 2 {
		return "", nil, errors.New("invalid auth header")
	}

	// check to see we have Bearer in auth-header
	if headerParts[0] != "Bearer" {
		return "", nil, errors.New("unauthorizedL no Bearer")
	}

	token := headerParts[1]

	// declare empty claims
	claims := &Claims{}

	// parse the token to out claims
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		// validate the signing method (alg)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected sigining method: %v", token.Header["alg"])
		}

		return []byte(app.JWTSecret), nil
	})

	// check for error, caught also expired tokens
	if err != nil {
		if strings.HasPrefix(err.Error(), "token is expired by") {
			return "", nil, errors.New("expired token")
		}

		return "", nil, err
	}

	// make sure we issued the token
	if claims.Issuer != app.Domain {
		return "", nil, errors.New("incorrect issuer")
	}

	// token is valid
	return token, claims, nil
}

func (app *application) generateTokenPair(user *data.User) (TokenPairs, error) {
	// create the token
	token := jwt.New(jwt.SigningMethodHS256)

	// set claims
	claims := token.Claims.(jwt.MapClaims)
	claims["name"] = fmt.Sprintf("%s %s", user.FirstName, user.LastName)
	claims["sub"] = fmt.Sprint(user.ID)
	claims["aud"] = app.Domain
	claims["iss"] = app.Domain

	if user.IsAdmin == 1 {
		claims["admin"] = true
	} else {
		claims["admin"] = false
	}

	// set the expiry
	claims["exp"] = time.Now().Add(jwtTokenExpiry).Unix()

	// create the signed token
	signedAccessToken, err := token.SignedString([]byte(app.JWTSecret))
	if err != nil {
		return TokenPairs{}, err
	}

	// create the refresh token
	refreshToken := jwt.New(jwt.SigningMethodHS256)
	refreshTokenClaims := refreshToken.Claims.(jwt.MapClaims)

	refreshTokenClaims["sub"] = fmt.Sprint(user.ID)
	refreshTokenClaims["aud"] = app.Domain
	refreshTokenClaims["iss"] = app.Domain

	// set expiry longer then refresh token
	refreshTokenClaims["exp"] = time.Now().Add(refreshTokenExpiry).Unix()

	// create signed refreshToken
	signedRefreshToken, err := refreshToken.SignedString([]byte(app.JWTSecret))
	if err != nil {
		return TokenPairs{}, err
	}

	var tokenPairs = TokenPairs{
		Token:        signedAccessToken,
		RefreshToken: signedRefreshToken,
	}

	return tokenPairs, nil
}
