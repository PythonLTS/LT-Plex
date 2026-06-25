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
	"syscall"
	"database/sql"
	"github.com/golang-jwt/jwt"
	"github.com/skip2/go-qrcode"
	"golang.org/x/crypto/bcrypt"
	_ "github.com/mattn/go-sqlite3"


	"fmt"
)

var base *sql.DB
var jwtKey = []byte("xxx2")

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

type StorageInfoResponse struct {
	Movies    int    `json:"movies"`
	Channels  int    `json:"channels"`
	Series    int    `json:"series"`
	FreeSpace string `json:"freeSpace"`
}

type FolderMeta struct {
	Name string `json:"Name"`
	Type string `json:"Type"`
}

func getDiskCapacity()string{

	var stat syscall.Statfs_t
	
	if err := syscall.Statfs("/", &stat); err != nil {
		log.Fatalf("Ошибка: %v", err)
	}

	// Считаем чистые байты -> переводим в Мегабайты
	freeBytes := stat.Bavail * uint64(stat.Bsize)
	freeMB := float64(freeBytes) / (1024 * 1024)

	// Фильтруем на лету для вывода в лог
	var finalLog string
	if freeMB >= 1024 {
		freeGB := freeMB / 1024
		finalLog = fmt.Sprintf("%.2f GB", freeGB)
		log.Println(finalLog)
		return finalLog
	} else {
		finalLog = fmt.Sprintf("%.2f MB", freeMB)
		log.Println(finalLog)
		return finalLog
	}

	// Выводим итоговый лог

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

	// Проверяем, существует ли папка. Если нет — создаем с правами 0755
	if _, err := os.Stat(root); os.IsNotExist(err) {
		err := os.MkdirAll(root, 0755)
		if err != nil {
			log.Println("Ошибка создания папки Films:", err)
			return nil
		}
		log.Println("Папка Films отсутствовала и была успешно создана")
	}

	// Читаем содержимое директории
	entries, err := ioutil.ReadDir(root)
	if err != nil {
		log.Println("Ошибка чтения папки Films:", err)
		return nil
	}

	list := make([]map[string]string, 0)

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		folder := e.Name()
		// Дефолтное изображение, если ничего не найдем
		logo := "/s/images/unknown.png" 

		files, _ := ioutil.ReadDir(root + "/" + folder)
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), ".png") {
				logo = "/films/" + folder + "/" + f.Name()
				break
			}
		}

		list = append(list, map[string]string{
			"id":   folder,                                   // ID для фронта (например, "Mr_Robot")
			"name": strings.ReplaceAll(folder, "_", " "),     // Красивое имя для вывода ("Mr Robot")
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
	if addMovie != "" {
		// Используем INSERT OR IGNORE, чтобы избежать дубликатов (благодаря UNIQUE-индексу)
		_, err := base.Exec("INSERT OR IGNORE INTO favorites (username, movie) VALUES (?, ?)", username, addMovie)
		if err != nil {
			log.Println("Ошибка добавления в избранное:", err)
		}
	}

	if removeMovie != "" {
		_, err := base.Exec("DELETE FROM favorites WHERE username = ? AND movie = ?", username, removeMovie)
		if err != nil {
			log.Println("Ошибка удаления из избранного:", err)
		}
	}

	// Получаем актуальный список избранного
	rows, err := base.Query("SELECT movie FROM favorites WHERE username = ?", username)
	var favs []string = make([]string, 0) // возвращаем пустой слайс вместо nil для JSON фронтенда
	if err != nil {
		log.Println("Ошибка получения списка избранного:", err)
		return favs
	}
	defer rows.Close()

	for rows.Next() {
		var movie string
		if err := rows.Scan(&movie); err == nil {
			favs = append(favs, movie)
		}
	}

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
	base, err = sql.Open("sqlite3", ".secure/Database.db")
	if err != nil {
		log.Fatal("Ошибка открытия БД:", err)
	}

	// Включаем поддержку Foreign Keys и создаем таблицы
	schema := `
	PRAGMA foreign_keys = ON;
	CREATE TABLE IF NOT EXISTS users (
		username TEXT PRIMARY KEY,
		password TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS favorites (
		username TEXT,
		movie TEXT,
		PRIMARY KEY (username, movie),
		FOREIGN KEY(username) REFERENCES users(username) ON DELETE CASCADE
	);`

	_, err = base.Exec(schema)
	if err != nil {
		log.Fatal("Ошибка создания таблиц:", err)
	}
}

func hashData(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("Ошибка хэширования пароля:", err)
		return ""
	}
	return string(hash)
}

func countContentByMeta(root string) (movies int, series int, channels int) {
	entries, err := os.ReadDir(root)
	if err != nil {
		log.Println("Ошибка чтения корневой директории контента:", err)
		return 0, 0, 0
	}

	for _, entry := range entries {
		// Нас интересуют только папки (например, Mr_Robot)
		if !entry.IsDir() {
			continue
		}

		// Путь к meta.json внутри этой папки
		metaPath := fmt.Sprintf("%s/%s/meta.json", root, entry.Name())

		// Проверяем, существует ли файл meta.json
		if _, err := os.Stat(metaPath); os.IsNotExist(err) {
			continue // Если мета-файла нет, пропускаем папку
		}

		// Читаем meta.json
		metaData, err := os.ReadFile(metaPath)
		if err != nil {
			log.Printf("Ошибка чтения файла %s: %v", metaPath, err)
			continue
		}

		// Парсим JSON
		var meta FolderMeta
		if err := json.Unmarshal(metaData, &meta); err != nil {
			log.Printf("Ошибка парсинга JSON в %s: %v", metaPath, err)
			continue
		}

		// Фильтруем и инкрементируем нужный счетчик
		switch strings.ToLower(meta.Type) {
		case "movie":
			movies++
		case "series":
			series++
		case "channel":
			channels++
		}
	}

	return movies, series, channels
}

func StorageInfo(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	mode := queryParams.Get("mode")

	if mode == "list" {
		// 1. Получаем свободное место на диске
		freeMemory := getDiskCapacity()

		// 2. Считаем контент по мета-файлам в "UserData/Films"
		moviesCount, seriesCount, channelsCount := countContentByMeta("UserData/Films")

		// 3. Собираем структуру ответа
		response := StorageInfoResponse{
			Movies:    moviesCount,
			Channels:  channelsCount,
			Series:    seriesCount,
			FreeSpace: freeMemory,
		}

		// 4. Отдаем JSON на фронтенд
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Твои логи для других режимов
	if mode == "series" {
		log.Println("Requested mode: series")
	}
	if mode == "movie" {
		log.Println("Requested mode: movie")
	}
	if mode == "channel" {
		log.Println("Requested mode: channel")
	}

	http.Error(w, "Missing or invalid 'mode' parameter.", http.StatusBadRequest)
}

func storageControl(w http.ResponseWriter,r *http.Request){
	http.ServeFile(w,r,"pages/StorageControl.html")
	
}

func addPack(w http.ResponseWriter,r *http.Request){

}

func deletePack(w http.ResponseWriter,r *http.Request){

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
	// Проверяем существование пользователя через COUNT
	err := base.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)", username).Scan(&exists)
	if err != nil {
		log.Println("Ошибка проверки пользователя:", err)
		return "error"
	}

	if exists {
		return "exists"
	}

	_, err = base.Exec("INSERT INTO users (username, password) VALUES (?, ?)", username, password)
	if err != nil {
		log.Println("Ошибка добавления пользователя:", err)
		return "error"
	}
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

	var storedHash string
	// Ищем хэш пароля пользователя в БД
	err := base.QueryRow("SELECT password FROM users WHERE username = ?", u.Username).Scan(&storedHash)
	if err == sql.ErrNoRows {
		http.Error(w, "Пользователь не найден", http.StatusUnauthorized)
		return
	} else if err != nil {
		http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(u.Password))
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

	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   86400,
		Secure:   true,
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
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
		http.Redirect(w, r, "/sign", http.StatusBadRequest)
		return
	}
	http.ServeFile(w, r, "pages/profile.html")
}

func home(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		http.ServeFile(w, r, "pages/index.html")
		return
	case "/sign":
		http.ServeFile(w, r, "pages/login.html")
		return
	default :
		http.Redirect(w,r,"/",http.StatusSeeOther)
		return
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
	http.HandleFunc("/api/addPackage", addPack)
	http.HandleFunc("/api/deletePackage", deletePack)
	http.HandleFunc("/api/StorageInfo", StorageInfo)
	http.HandleFunc("/StorageControl", storageControl)
	http.HandleFunc("/qrdata/", pushqr)
	http.HandleFunc("/data-confirm/", qrConfirm)
	http.HandleFunc("/data-status/", qrStatus)
	http.HandleFunc("/profile", mainprofile)
	http.HandleFunc("/profile-data", profileDataHandler)
	http.HandleFunc("/api/logout", logoutHandler)
	http.HandleFunc("/addfv", addFavoriteHandler)
	http.HandleFunc("/rmfv", removeFavoriteHandler)
	http.HandleFunc("/films", filmsHandler)
	http.Handle("/films/", http.StripPrefix("/films/", http.FileServer(http.Dir("UserData/Films"))))
	http.Handle("/s/", http.StripPrefix("/s/", http.FileServer(http.Dir("static"))))

	go cleanupQRSessions()
	log.Println("started")
	http.ListenAndServeTLS(":443", ".secure/certs/server.crt", ".secure/certs/server.key", nil)
}