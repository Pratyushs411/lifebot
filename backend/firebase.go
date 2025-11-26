package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	firebase "firebase.google.com/go"
	"google.golang.org/api/option"
)

var fbClient *firebase.App

// ⚠️ IMPORTANT: Update this to your Firebase RTDB URL
// Example: https://lifebot-default-rtdb.firebaseio.com/
var fbDBURL = "https://lifebot-b28a9.firebaseio.com/"

// Initialize Firebase
func InitFirebase(ctx context.Context) error {
	opt := option.WithCredentialsFile("serviceAccount.json")

	app, err := firebase.NewApp(ctx, &firebase.Config{
		DatabaseURL: fbDBURL,
	}, opt)
	if err != nil {
		return fmt.Errorf("Firebase init failed: %w", err)
	}

	fbClient = app
	fmt.Println("🔥 Firebase Realtime Database connected")
	return nil
}

// PUT request to Firebase
func FirebaseSet(path string, v interface{}) error {
	url := fbDBURL + path + ".json"

	jsonBody, err := json.Marshal(v)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	_, err = client.Do(req)
	return err
}

// GET request to Firebase
func FirebaseGet(path string, out interface{}) error {
	url := fbDBURL + path + ".json"

	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return json.NewDecoder(res.Body).Decode(out)
}
