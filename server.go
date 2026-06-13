package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync" // Добавлено для потокобезопасности
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/skip2/go-qrcode"
	"go.etcd.io/bbolt"
	"golang.org/x/crypto/bcrypt"
)

var base *bbolt.DB
var jwtKey = []byte("LT-Security-Laboratory-SecretKey:awd9a0w120saplcaw/wqd")

type QRSession struct {
	Token      string
	Username   string
	Authorized bool
	CreatedAt  time.Time
}

// Мапа сессий и мьютекс для защиты от race condition
var qrSessions = make(map[string]*QRSession)
var qrMutex sync.RWMutex

type User struct {
	Username string `json:"Username"`
	Password string `json:"Password"`
}

type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

func allFilmsPage(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("jwt")
	if err != nil {
		http.Redirect(w, r, "/sign", http.StatusSeeOther)
		return
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(cookie.Value, claims, func(t *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		http.Redirect(w, r, "/sign", http.StatusSeeOther)
		return
	}

	http.ServeFile(w, r, "pages/filmList.html")
}

func scanFilms() []map[string]string {
	root := "Films"
	entries, err := ioutil.ReadDir(root)
	if err != nil {
		return nil
	}

	var list []map[string]string

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		folder := e.Name()
		logo := ""

		files, _ := ioutil.ReadDir(root + "/" + folder)
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), ".png") {
				logo = "/films/" + folder + "/" + f.Name()
				break
			}
		}

		list = append(list, map[string]string{
			"name": folder,
			"logo": logo,
		})
	}

	return list
}

func filmsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("filmsHandler")
	cookie, err := r.Cookie("jwt")
	log.Println("cookie: ",cookie)
	if err != nil {
		http.Error(w, "неавторизован", http.StatusUnauthorized)
		return
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(cookie.Value, claims, func(t *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		http.Redirect(w, r, "/sign", http.StatusSeeOther)
		return
	}
	list := scanFilms()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func favorites(username string, addMovie, removeMovie string) []string {
	var favs []string
	base.Update(func(tx *bbolt.Tx) error {
		b, _ := tx.CreateBucketIfNotExists([]byte("Favorites"))
		raw := b.Get([]byte(username))

		if raw != nil {
			json.Unmarshal(raw, &favs)
		}

		changed := false

		if addMovie != "" {
			exists := false
			for _, f := range favs {
				if f == addMovie {
					exists = true
					break
				}
			}
			if !exists {
				favs = append(favs, addMovie)
				changed = true
			}
		}

		if removeMovie != "" {
			for i := len(favs) - 1; i >= 0; i-- {
				if favs[i] == removeMovie {
					favs = append(favs[:i], favs[i+1:]...)
					changed = true
					break
				}
			}
		}

		if changed {
			data, _ := json.Marshal(favs)
			b.Put([]byte(username), data)
		}

		return nil
	})

	return favs
}

func addFavoriteHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("jwt")
	if err != nil {
		http.Error(w, "unauthorized", 401)
		return
	}

	claims := &Claims{}
	_, err = jwt.ParseWithClaims(cookie.Value, claims, func(t *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		http.Error(w, "unauthorized", 401)
		return
	}

	var body struct {
		Movie string `json:"movie"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	favorites(claims.Username, body.Movie, "")

	w.Write([]byte("ok"))
}

func removeFavoriteHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("jwt")
	if err != nil {
		http.Error(w, "unauthorized", 401)
		return
	}

	claims := &Claims{}
	_, err = jwt.ParseWithClaims(cookie.Value, claims, func(t *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		http.Error(w, "unauthorized", 401)
		return
	}

	var body struct {
		Movie string `json:"movie"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	favorites(claims.Username, "", body.Movie)

	w.Write([]byte("ok"))
}

func profileDataHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("jwt")
	if err != nil {
		http.Error(w, "неавторизован", http.StatusUnauthorized)
		return
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(cookie.Value, claims, func(t *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		http.Error(w, "неавторизован", http.StatusUnauthorized)
		return
	}

	favs := favorites(claims.Username, "", "")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"username":  claims.Username,
		"avatar":    "/s/images/default_avatar.png",
		"favorites": favs,
	})
}

func qrStatus(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Потокобезопасное чтение мапы сессий
	qrMutex.RLock()
	session, ok := qrSessions[token]
	qrMutex.RUnlock()

	if ok && session.Authorized {
		jwtToken, err := generateToken(session.Username)
		if err != nil {
			http.Error(w, "token error", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "jwt",
			Value:    jwtToken,
			Path:     "/",
			MaxAge:   7200,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   true,
		})

		// Потокобезопасное удаление из мапы
		qrMutex.Lock()
		delete(qrSessions, token)
		qrMutex.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "pending"})
}

func qrConfirm(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	cookie, err := r.Cookie("jwt")
	if err != nil {
		http.Error(w, "неавторизован", http.StatusUnauthorized)
		return
	}

	claims := &Claims{}
	jwtToken, err := jwt.ParseWithClaims(cookie.Value, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil || !jwtToken.Valid {
		http.Error(w, "неавторизован", http.StatusUnauthorized)
		return
	}

	// Потокобезопасное изменение сессии
	qrMutex.Lock()
	session, ok := qrSessions[token]
	if ok {
		session.Authorized = true
		session.Username = claims.Username
	}
	qrMutex.Unlock()

	if ok {
		w.Write([]byte("ok"))
	} else {
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func pushqr(w http.ResponseWriter, r *http.Request) {
	token := strconv.FormatInt(time.Now().UnixNano(), 10) + "_" + strconv.Itoa(rand.Intn(100000))

	// Потокобезопасное добавление сессии
	qrMutex.Lock()
	qrSessions[token] = &QRSession{
		Token:     token,
		CreatedAt: time.Now(),
	}
	qrMutex.Unlock()

	url := "https://lteam.sec/data-confirm?token=" + token

	png, err := qrcode.Encode(url, qrcode.Medium, 256)
	if err != nil {
		http.Error(w, "qr error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("X-QR-Token", token)
	w.Write(png)
}

func init_DB() {
	os.MkdirAll(".secure", 0755)
	var err error
	base, err = bbolt.Open(".secure/Database.db", 0666, nil)
	if err != nil {
		return
	}
	base.Update(func(tx *bbolt.Tx) error {
		tx.CreateBucketIfNotExists([]byte("Users"))
		tx.CreateBucketIfNotExists([]byte("Favorites"))
		return nil
	})
}

func hashData(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("Ошибка хэширования пароля:", err)
		return ""
	}
	return string(hash)
}

func generateToken(username string) (string, error) {
	claims := Claims{
		Username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func adduser(username, password string) string {
	var exists bool
	base.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("Users"))
		if b.Get([]byte(username)) != nil {
			exists = true
		}
		return nil
	})

	if exists {
		return "exists"
	}

	base.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("Users"))
		b.Put([]byte(username), []byte(password))
		return nil
	})
	return "ok"
}

func getdataLogin(w http.ResponseWriter, r *http.Request) {
	log.Println("api Login accessed")
	defer r.Body.Close()
	var u User
	json.NewDecoder(r.Body).Decode(&u)

	if u.Username == "" || u.Password == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var storedHash []byte
	base.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("Users"))
		storedHash = b.Get([]byte(u.Username))
		return nil
	})

	if storedHash == nil {
		http.Error(w, "Пользователь не найден", http.StatusUnauthorized)
		return
	}

	err := bcrypt.CompareHashAndPassword(storedHash, []byte(u.Password))
	if err != nil {
		http.Error(w, "Неверный Логин или Пароль", http.StatusUnauthorized)
		return
	}

	token, err := generateToken(u.Username)
	if err != nil {
		http.Error(w, "Token Error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   86400,
		Secure:   true,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func getdataRegister(w http.ResponseWriter, r *http.Request) {
	log.Println("api register accessed")
	defer r.Body.Close()
	var u User
	json.NewDecoder(r.Body).Decode(&u)

	if u.Username == "" || u.Password == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	hashed := hashData(u.Password)
	if hashed == "" {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	result := adduser(u.Username, hashed)

	if result == "exists" {
		log.Println("имя уже занято")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("exists"))
		return
	}

	token, err := generateToken(u.Username)
	if err != nil {
		http.Error(w, "token error", http.StatusInternalServerError)
		return
	}

	// ИСПРАВЛЕНО: Сразу авторизуем пользователя через куку при успешной регистрации
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   86400,
		Secure:   true,
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok")) // Фронтенд считает "ok" и поймет, что всё успешно
}

func mainprofile(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("jwt")
	if err != nil {
		http.Redirect(w, r, "/sign", http.StatusSeeOther)
		return
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(cookie.Value, claims, func(t *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		http.Redirect(w, r, "/sign", http.StatusSeeOther)
		return
	}
	http.ServeFile(w, r, "pages/profile.html")
}

func home(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		http.ServeFile(w, r, "pages/index.html")
	case "/sign":
		http.ServeFile(w, r, "pages/login.html")
	}
}

func licenseHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "pages/license.html")
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func cleanupQRSessions() {
	for {
		time.Sleep(1 * time.Minute)
		now := time.Now()

		// Потокобезопасная очистка старых сессий
		qrMutex.Lock()
		for k, v := range qrSessions {
			if now.Sub(v.CreatedAt) > 5*time.Minute {
				delete(qrSessions, k)
			}
		}
		qrMutex.Unlock()
	}
}

func main() {
	init_DB()
	defer base.Close()

	http.HandleFunc("/filmsPage", allFilmsPage)
	http.HandleFunc("/", home)
	http.HandleFunc("/l/", licenseHandler)
	http.HandleFunc("/api/registerData/", getdataRegister)
	http.HandleFunc("/api/loginData/", getdataLogin)
	http.HandleFunc("/qrdata/", pushqr)
	http.HandleFunc("/data-confirm/", qrConfirm)
	http.HandleFunc("/data-status/", qrStatus)
	http.HandleFunc("/profile", mainprofile)
	http.HandleFunc("/profile-data", profileDataHandler)
	http.HandleFunc("/lt", logoutHandler)
	http.HandleFunc("/addfv", addFavoriteHandler)
	http.HandleFunc("/rmfv", removeFavoriteHandler)
	http.HandleFunc("/films", filmsHandler)

	http.Handle("/films/", http.StripPrefix("/films/", http.FileServer(http.Dir("Films"))))
	http.Handle("/s/", http.StripPrefix("/s/", http.FileServer(http.Dir("static"))))

	go cleanupQRSessions()
	log.Println("started")
	http.ListenAndServeTLS(":443", ".secure/certs/server.crt", ".secure/certs/server.key", nil)
}