package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"

	"github.com/buzzapp/user/model"
	"github.com/buzzapp/user/reqres"
)

func handleCreateUser(svc UserService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the body into a string for json decoding
		var payload = &reqres.CreateUserRequest{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			respondWithError("unable to decode json request", err, w, http.StatusInternalServerError)
			return
		}

		// Do some validation
		if err := validateCreateUser(payload); err != nil {
			respondWithError("Validation error", err, w, http.StatusBadRequest)
			return
		}

		// Create our new user struct
		newUser := &model.CreateUser{
			Email:     payload.Email,
			FirstName: payload.FirstName,
			LastName:  payload.LastName,
			Password:  payload.Password,
			Role:      payload.Role,
			Username:  payload.Username,
		}

		// save the app to our database
		user, err := svc.Create(newUser)
		if err != nil {
			respondWithError("unable to add user", err, w, http.StatusInternalServerError)
			return
		}

		// Generate our response
		resp := reqres.CreateUserResponse{User: user}

		// Marshal up the json response
		js, err := json.Marshal(resp)
		if err != nil {
			respondWithError("unable to marshal json response", err, w, http.StatusInternalServerError)
			return
		}

		// Return the response
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	})
}

func handleGetUserByID(svc UserService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the user ID from the url
		id := mux.Vars(r)["id"]

		// Do some validation
		if err := validateGetUserByID(id); err != nil {
			respondWithError("Validation error", err, w, http.StatusBadRequest)
			return
		}

		// get the user from our database
		user, err := svc.GetByID(id)
		if err != nil {
			respondWithError("unable to get user", err, w, http.StatusInternalServerError)
			return
		}

		// Generate our response
		resp := reqres.GetUserResponse{User: user}

		// Marshal up the json response
		js, err := json.Marshal(resp)
		if err != nil {
			respondWithError("unable to marshal json response", err, w, http.StatusInternalServerError)
			return
		}

		// Return the response
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	})
}

func handleLoginUser(svc UserService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the body into a string for json decoding
		var payload = &reqres.LoginRequest{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			respondWithError("unable to decode json request", err, w, http.StatusInternalServerError)
			return
		}

		// Do some validation
		if err := validateLoginUser(payload); err != nil {
			respondWithError("Validation error", err, w, http.StatusBadRequest)
			return
		}

		// save the app to our database
		token, err := svc.Login(payload.Username, payload.Password, r.Referer())
		if err != nil {
			respondWithError("unable to log in user", err, w, http.StatusBadRequest)
			return
		}

		// Generate our response
		resp := reqres.LoginResponse{Token: token}

		// Marshal up the json response
		js, err := json.Marshal(resp)
		if err != nil {
			respondWithError("unable to marshal json response", err, w, http.StatusInternalServerError)
			return
		}

		// Return the response
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	})
}

func handleRefreshToken(svc UserService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the body into a string for json decoding
		var payload = &reqres.RefreshTokenRequest{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			respondWithError("unable to decode json request", err, w, http.StatusInternalServerError)
			return
		}

		if err := validateRefreshToken(payload); err != nil {
			respondWithError("Validation error", err, w, http.StatusBadRequest)
			return
		}

		// Decode jwt token
		token, err := jwt.Parse(payload.Token, func(token *jwt.Token) (interface{}, error) {
			// Valid alg is what we expect
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(SecretKey), nil
		})
		if err != nil {
			respondWithError("Access not allowed", err, w, http.StatusForbidden)
			return
		}

		if !token.Valid {
			respondWithError("Access not allowed", errors.New("Invalid jwt token"), w, http.StatusForbidden)
			return
		}

		jwtToken, err := svc.RefreshToken(token.Claims["sub"].(string), token.Claims["username"].(string), token.Claims["role"].(string), r.Referer())
		if err != nil {
			respondWithError("unable to refresh token", err, w, http.StatusInternalServerError)
			return
		}

		// Generate our response
		resp := reqres.LoginResponse{Token: jwtToken}

		// Marshal up the json response
		js, err := json.Marshal(resp)
		if err != nil {
			respondWithError("unable to marshal json response", err, w, http.StatusInternalServerError)
			return
		}

		// Return the response
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	})
}

// Helper function to return a json error message
func respondWithError(msg string, err error, w http.ResponseWriter, status int) {
	errMsg := reqres.ErrorResponse{Message: msg + ": " + err.Error()}

	js, err := json.Marshal(errMsg)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)
}
