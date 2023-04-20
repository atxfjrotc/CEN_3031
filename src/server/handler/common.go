package handler

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	// "github.com/joho/godotenv"
	"github.com/atxfjrotc/uswap/src/server/db"
	"github.com/atxfjrotc/uswap/src/server/utils"
	"github.com/golang-jwt/jwt/v5"
)

func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:4200")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type")
			w.Header().Set("Content-Type", "application/json")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "http://localhost:4200")
}

type Login struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

var jwtKey = []byte(os.Getenv("RSA_PRIVATE_KEY"))

func LoginPost(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	var login Login
	json.Unmarshal(body, &login)

	ctx := db.Ctx
	er := db.DB.PingContext(ctx)
	if er != nil {
		panic(er)
	}

	tsql := fmt.Sprintf("SELECT Password FROM TestSchema.Users WHERE Name = @Name")

	rows, err := db.DB.QueryContext(ctx, tsql, sql.Named("Name", login.Username))
	if err != nil {
		fmt.Println("Error with creating db query")
		panic(err)
	}
	defer rows.Close()

	var hash string

	for rows.Next() {
		err := rows.Scan(&hash)
		if err != nil {
			log.Fatal(err)
		}
	}

	success := utils.CheckPasswordHash(login.Password, string(hash))

	if !success {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	id_rows, err := db.DB.Query("SELECT user_id FROM users WHERE user_name = ?", login.Username)

	if err != nil {
		fmt.Println("Error when selecting id query")
		log.Fatal(err)
	}
	defer id_rows.Close()

	var sub string

	for id_rows.Next() {
		if err := id_rows.Scan(&sub); err != nil {
			log.Fatal(err)
		}
	}

	expirationTime := time.Now().Add(time.Minute)

	claims := &Claims{
		Username: login.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: sub,
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println(err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:    "token",
		Value:   tokenString,
		Expires: expirationTime,
	})

	loginJson, err := json.Marshal(login)
	if err != nil {
		panic(err)
	}

	var m map[string]interface{}
	json.Unmarshal(loginJson, &m)
	m["id_token"] = sub
	response, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

type SignUp struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func SignUpPost(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	var signup SignUp
	json.Unmarshal(body, &signup)

	createID, err := db.CreateUser(signup.Username, signup.Email, signup.Password)
	if err != nil {
		log.Fatal("Error creating User: ", err.Error())
	}
	fmt.Printf("Inserted ID: %d successfully.\n", createID)

	w.WriteHeader(http.StatusOK)
}

type Item struct {
	Name        string `json:"itemName"`
	Description string `json:"itemDescription"`
	UserID      string `json:"userID"`
	ImagePath   []byte `json:"image"`
}

func CreateListing(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
        return
	}

	defer r.Body.Close()

	image := r.FormValue("imageSrc")

    // Decode the base64 encoded string into image data
	imageData, err := base64.StdEncoding.DecodeString(strings.TrimSpace(image))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Write the image data to a local file
	err = ioutil.WriteFile("image.jpg", imageData, 0644)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

    // send response back to client
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Item created successfully"))
}

type ItemID struct {
	ItemID string `json:"itemID"`
}

func GetItem(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	var itemID ItemID
	json.Unmarshal(body, &itemID)

	item, err := db.GetItem(itemID.ItemID)

	jsonBytes, err := utils.StructToJSON(item)
	if err != nil {
		fmt.Print(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonBytes)
}

func SearchItems(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	type Search struct {
		Search string `json:"search"`
	}
	var search Search
	json.Unmarshal(body, &search)

	items, _ := db.SearchItems(search.Search)

	jsonBytes, err := utils.StructToJSON(items)
	if err != nil {
		fmt.Print(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonBytes)

}

func GetItems(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	items, _ := db.GetItems()

	jsonBytes, err := utils.StructToJSON(items)
	if err != nil {
		fmt.Print(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonBytes)

}

type Swap struct {
	SenderID       string `json:"senderID"`
	SenderItemID   string `json:"senderItemID"`
	ReceiverID     string `json:"receiverID"`
	ReceiverItemID string `json:"receiverItemID"`
}

func CreateSwapRequest(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	var swap Swap
	json.Unmarshal(body, &swap)

	_, err = db.CreateSwapRequest(swap.SenderID, swap.SenderItemID, swap.ReceiverID, swap.ReceiverItemID)
	if err != nil {
		log.Fatal("Failed to create the swap request")
	}
}

type SwapID struct {
	SwapID string `json:"swapID"`
}

func GetSwapRequest(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	var swapID SwapID
	json.Unmarshal(body, &swapID)

	swapRequest, err := db.GetItem(swapID.SwapID)

	jsonBytes, err := utils.StructToJSON(swapRequest)
	if err != nil {
		fmt.Print(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonBytes)
}

func AcceptSwapRequest(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	var swapID SwapID
	json.Unmarshal(body, &swapID)

	err = db.AcceptSwapRequest(swapID.SwapID)
	if err != nil {
		log.Panic("Failed to accept swap request")
	}
}

func RejectSwapRequest(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	var swapID SwapID
	json.Unmarshal(body, &swapID)

	err = db.RejectSwapRequest(swapID.SwapID)
	if err != nil {
		log.Panic("Failed to accept swap request")
	}
}
