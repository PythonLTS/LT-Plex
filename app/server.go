package main

import (
	"encoding/json"
	"io/ioutil"
	"io"
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
	"path/filepath"
	"archive/zip"
	"html/template"

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
	Quantity int `json:"Quantity"`
}

type FileItemResponse struct {
	Name string `json:"name"`
	Logo string `json:"logo"`
}

type MetaData struct {
	Name              string   `json:"Name"`
	Type              string   `json:"Type"` // series, movie, channel
	Quantity          int      `json:"Quantity,omitempty"`
	Seasons           int      `json:"Seasons,omitempty"`
	EpisodesPerSeason []int    `json:"EpisodesPerSeason,omitempty"`
}

type TemplateData struct {
	Name       string
	FolderAttr string // Имя папки для URL (например, Mr_Robot)
	Meta       MetaData
	Videos     []string // Только для каналов: список реальных имен файлов
}

func getContentListByType(root string, targetType string) []FileItemResponse {
	list := make([]FileItemResponse, 0)
	entries, err := os.ReadDir(root)
	if err != nil {
		return list
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		folder := entry.Name()
		metaPath := fmt.Sprintf("%s/%s/meta.json", root, folder)
		metaData, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}

		var meta FolderMeta
		if err := json.Unmarshal(metaData, &meta); err != nil {
			continue
		}

		if strings.ToLower(meta.Type) == strings.ToLower(targetType) {
			// Формируем путь к обложке: /films/Имя_Папки/Имя_Папки.jpeg
			// (Твой статический роут в main раздает UserData/Films через префикс /films/)
			logoPath := fmt.Sprintf("/films/%s/%s.jpeg", folder, folder)
			
			// Проверим физически, есть ли файл на диске, если нет — ставим заглушку
			if _, err := os.Stat(fmt.Sprintf("%s/%s/%s.jpeg", root, folder, folder)); os.IsNotExist(err) {
				logoPath = "/s/images/unknown.png"
			}

			list = append(list, FileItemResponse{
				Name: folder,
				Logo: logoPath,
			})
		}
	}
	return list
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
	root := "UserData/Films"

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
			if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), ".jpeg") {
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
	
	rootPath := "UserData/Films"

	// Режим 1: Общая статистика для главного экрана
	if mode == "list" {
		freeMemory := getDiskCapacity()
		moviesCount, seriesCount, channelsCount := countContentByMeta(rootPath)

		response := StorageInfoResponse{
			Movies:    moviesCount,
			Channels:  channelsCount,
			Series:    seriesCount,
			FreeSpace: freeMemory,
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Режим 2: Запрос конкретного контента для модалки по клику
	if mode == "movie" || mode == "series" || mode == "channel" {
		// Собираем слайс структур FileItemResponse
		filesList := getContentListByType(rootPath, mode)
		
		// Если ничего не нашли, отдаем пустой массив [], а не null, чтобы JS не падал
		if filesList == nil {
			filesList = make([]FileItemResponse, 0)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(filesList)
		return
	}

	// Если mode не подошёл ни под один критерий
	http.Error(w, "Missing or invalid 'mode' parameter.", http.StatusBadRequest)
}

func storageControl(w http.ResponseWriter,r *http.Request){
	http.ServeFile(w,r,"pages/StorageControl.html")
	
}

func GenerateHTML(meta MetaData, folderName string, packDir string) (string, error) {
	var rawTemplate string

	switch meta.Type {
	case "movie":
		rawTemplate = tmplMovie
	case "series":
		rawTemplate = tmplSeries
	case "channel":
		rawTemplate = tmplChannel
	default:
		return "", nil
	}

	// Для каналов собираем реальные имена файлов из папки
	var videoFiles []string
	if meta.Type == "channel" {
		files, err := os.ReadDir(packDir)
		if err == nil {
			for _, f := range files {
				if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), ".mp4") {
					videoFiles = append(videoFiles, f.Name())
				}
			}
		}
	}

	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"seq": func(start, end int) []int {
			s := make([]int, end-start+1)
			for i := range s {
				s[i] = start + i
			}
			return s
		},
		// Хелпер для красивого отображения имени видео (убирает .mp4 и заменяет _ на пробел)
		"cleanName": func(filename string) string {
			name := strings.TrimSuffix(filename, filepath.Ext(filename))
			return strings.ReplaceAll(name, "_", " ")
		},
	}

	tmpl, err := template.New("player").Funcs(funcMap).Parse(rawTemplate)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	data := TemplateData{
		Name:       strings.ReplaceAll(meta.Name, "_", " "), // Красивое имя для заголовка
		FolderAttr: folderName,                              // Имя папки для URL
		Meta:       meta,
		Videos:     videoFiles,
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func addPack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fileName := r.Header.Get("X-File-Name")
	if fileName == "" {
		http.Error(w, "missing file name", http.StatusBadRequest)
		return
	}

	fileName = filepath.Base(fileName)
	baseName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	packDir := filepath.Join("UserData/Films", baseName)

	if err := os.MkdirAll(packDir, 0755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	zipPath := filepath.Join(packDir, fileName)
	zipFile, err := os.Create(zipPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer zipFile.Close()

	if _, err := io.Copy(zipFile, r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	for _, f := range reader.File {
		targetPath := filepath.Join(packDir, f.Name)

		if !strings.HasPrefix(
			filepath.Clean(targetPath),
			filepath.Clean(packDir)+string(os.PathSeparator),
		) {
			continue
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, os.ModePerm); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), os.ModePerm); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		src, err := f.Open()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		dst, err := os.OpenFile(
			targetPath,
			os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
			f.Mode(),
		)
		if err != nil {
			src.Close()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = io.Copy(dst, src)
		dst.Close()
		src.Close()

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	_ = os.Remove(zipPath)

	metaPath := filepath.Join(packDir, "meta.json")
	metaFile, err := os.Open(metaPath)
	if err != nil {
		http.Error(w, "meta.json not found in archive", http.StatusBadRequest)
		return
	}
	defer metaFile.Close()

	var meta MetaData
	if err := json.NewDecoder(metaFile).Decode(&meta); err != nil {
		http.Error(w, "invalid meta.json format: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Передаем baseName (имя папки) и packDir для сканирования файлов канала
	htmlContent, err := GenerateHTML(meta, baseName, packDir)
	if err != nil {
		http.Error(w, "failed to generate HTML: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Сохраняем файл как Имя_Архива.html
	indexPath := filepath.Join(packDir, baseName+".html")
	if err := os.WriteFile(indexPath, []byte(htmlContent), 0644); err != nil {
		http.Error(w, "failed to save html: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("done"))
}

const tmplMovie = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>{{.Name}}</title>
<style>
:root{--bg:#0a0a0a;--panel:#141414;--panel2:#1b1b1b;--border:#2a2a2a;--text:#ffffff;--muted:#9a9a9a;}
*{margin:0;padding:0;box-sizing:border-box;font-family:system-ui;}
body{background:var(--bg);color:var(--text);}
#topbar{display:flex;align-items:center;gap:14px;padding:14px 20px;background:var(--panel);border-bottom:1px solid var(--border);}
#back{background:var(--panel2);border:1px solid var(--border);padding:8px 14px;border-radius:10px;color:white;cursor:pointer;}
#content{display:flex;height:calc(100vh - 60px);}
#info{width:320px;background:var(--panel);border-right:1px solid var(--border);padding:20px;}
#info h2{margin-bottom:8px;}
#info p{color:var(--muted);margin-top:10px;}
#player{flex:1;display:flex;align-items:center;justify-content:center;padding:20px;}
video{width:100%;height:100%;border-radius:14px;background:black;box-shadow:0 0 30px rgba(0,0,0,0.6);}
</style>
</head>
<body>
<div id="topbar">
    <button id="back" onclick="window.history.back()">← Back</button>
    <h1>{{.Name}}</h1>
</div>
<div id="content">
    <div id="info">
        <h2>{{.Name}}</h2>
        <p>Movie</p>
    </div>
    <div id="player">
        <video src="/films/{{.FolderAttr}}/{{.FolderAttr}}.mp4" controls></video>
    </div>
</div>
</body>
</html>`

const tmplSeries = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>{{.Name}}</title>
<style>
:root{--bg:#0a0a0a;--panel:#141414;--panel2:#1b1b1b;--border:#2a2a2a;--text:#fff;--muted:#9a9a9a;--accent:#e5a00d;}
*{margin:0;padding:0;box-sizing:border-box;font-family:system-ui;}
body{background:var(--bg);color:var(--text);}
#topbar{padding:14px 20px;background:var(--panel);border-bottom:1px solid var(--border);display:flex;gap:14px;align-items:center;}
#back{background:var(--panel2);border:1px solid var(--border);padding:8px 14px;border-radius:10px;color:white;cursor:pointer;}
#content{display:flex;height:calc(100vh - 60px);}
#sidebar{width:340px;background:var(--panel);border-right:1px solid var(--border);overflow:auto;padding:16px;}
.season{margin-bottom:10px;}
.season-header{background:var(--panel2);padding:12px;border-radius:10px;cursor:pointer;border:1px solid var(--border);}
.season-header:hover, .season-header.active{background:#222;}
.episodes{list-style:none;margin-top:8px;padding-left:10px;display:none;}
.episodes li{padding:10px;margin:6px 0;background:var(--panel2);border:1px solid var(--border);border-radius:10px;cursor:pointer;color:var(--muted);}
.episodes li:hover{background:#222;color:white;}
.episodes li.active{background:var(--accent);color:black;font-weight:600;border:none;}
#player{flex:1;display:flex;justify-content:center;align-items:center;padding:20px;}
video{width:100%;height:100%;border-radius:14px;background:black;box-shadow:0 0 30px rgba(0,0,0,0.6);}
</style>
</head>
<body>
<div id="topbar">
    <button id="back" onclick="window.history.back()">← Back</button>
    <h1>{{.Name}}</h1>
</div>
<div id="content">
    <div id="sidebar">
        {{range $index, $epCount := .Meta.EpisodesPerSeason}}
        {{$seasonNum := add $index 1}}
        <div class="season" data-season="{{$seasonNum}}">
            <div class="season-header">Season {{$seasonNum}}</div>
            <ul class="episodes">
                {{range $ep := seq 1 $epCount}}
                <li data-ep="{{$ep}}">Episode {{$ep}}</li>
                {{end}}
            </ul>
        </div>
        {{end}}
    </div>
    <div id="player">
        <video id="vplayer" controls></video>
    </div>
</div>
<script>
document.addEventListener("DOMContentLoaded", () => {
    const seasons = document.querySelectorAll(".season");
    
    seasons.forEach(season => {
        const header = season.querySelector(".season-header");
        const list = season.querySelector(".episodes");

        header.addEventListener("click", () => {
            document.querySelectorAll(".episodes").forEach(l => {
                if (l !== list) l.style.display = "none";
            });
            document.querySelectorAll(".season-header").forEach(h => h.classList.remove("active"));
            
            header.classList.add("active");
            list.style.display = (list.style.display === "block") ? "none" : "block";
        });

        season.querySelectorAll("li").forEach(ep => {
            ep.addEventListener("click", (e) => {
                e.stopPropagation();
                document.querySelectorAll(".episodes li").forEach(li => li.classList.remove('active'));
                ep.classList.add('active');

                const sNum = season.dataset.season;
                const eNum = ep.dataset.ep;
                
                const video = document.getElementById('vplayer');
                video.src = "/films/{{.FolderAttr}}/Season" + sNum + "/Episode" + eNum + ".mp4";
                video.load();
                video.play();
            });
        });
    });
});
</script>
</body>
</html>`

const tmplChannel = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>{{.Name}}</title>
<style>
:root{--bg:#0a0a0a;--panel:#141414;--panel2:#1b1b1b;--border:#2a2a2a;--text:#fff;--muted:#9a9a9a;}
*{margin:0;padding:0;box-sizing:border-box;font-family:system-ui;}
body{background:var(--bg);color:var(--text);}
#topbar{padding:14px;background:var(--panel);border-bottom:1px solid var(--border);display:flex;gap:14px;align-items:center;}
#back{background:var(--panel2);border:1px solid var(--border);padding:8px 14px;border-radius:10px;color:white;cursor:pointer;}
#content{display:flex;height:calc(100vh - 60px);}
#sidebar{width:340px;background:var(--panel);border-right:1px solid var(--border);padding:14px;overflow:auto;}
#search{width:100%;padding:10px;border-radius:10px;border:1px solid var(--border);background:var(--panel2);color:white;outline:none;margin-bottom:12px;}
.video-item{padding:12px;margin:6px 0;background:var(--panel2);border:1px solid var(--border);border-radius:10px;cursor:pointer;color:var(--muted);}
.video-item:hover{background:#222;color:white;}
.video-item.active{background:#2a2a2a;color:#fff;border-color:#555;}
#player{flex:1;display:flex;justify-content:center;align-items:center;padding:20px;flex-direction:column;gap:10px;}
video{width:100%;height:100%;border-radius:14px;background:black;box-shadow:0 0 30px rgba(0,0,0,0.6);}
#error-log{color:#ff4a4a;font-size:14px;display:none;background:rgba(255,0,0,0.1);padding:10px;border-radius:8px;width:100%;text-align:center;}
</style>
</head>
<body>
<div id="topbar">
    <button id="back" onclick="window.history.back()">← Back</button>
    <h1 class="movie-title">{{.Name}}</h1>
</div>
<div id="content">
    <div id="sidebar">
        <input id="search" placeholder="Search videos..." oninput="filterVideos()">
        <div id="video-list">
            {{range $file := .Videos}}
            <div class="video-item" data-src="{{$file}}">{{cleanName $file}}</div>
            {{end}}
        </div>
    </div>
    <div id="player">
        <div id="error-log"></div>
        <video id="vplayer" controls></video>
    </div>
</div>
<script>
document.addEventListener("DOMContentLoaded", () => {
    const video = document.getElementById('vplayer');
    const errLog = document.getElementById('error-log');

    // Отслеживаем ошибки видео-плеера для дебага
    video.addEventListener('error', () => {
        errLog.style.display = "block";
        if (video.error) {
            switch (video.error.code) {
                case 1: errLog.textContent = "Загрузка прервана пользователем."; break;
                case 2: errLog.textContent = "Ошибка сети при загрузке видео."; break;
                case 3: errLog.textContent = "Ошибка декодирования видео (битый кодек/файл)."; break;
                case 4: errLog.textContent = "Видео не найдено по указанному пути (404 Not Found). Текущий URL: " + video.src; break;
                default: errLog.textContent = "Неизвестная ошибка плеера."; break;
            }
        }
    });

    document.querySelectorAll(".video-item").forEach(item => {
        item.addEventListener("click", () => {
            document.querySelectorAll(".video-item").forEach(i => i.classList.remove('active'));
            item.classList.add('active');
            errLog.style.display = "none";

            const file = item.dataset.src;
            
            // Убрали encodeURIComponent. Передаем чистую строку, браузер сам ее экранирует для HTTP-запроса.
            // Путь строится как: /films/Имя_Папки/Имя_Файла.mp4
            video.src = "/films/{{.FolderAttr}}/" + file;
            video.load();
            video.play().catch(err => {
                console.log("Автозапуск заблокирован браузером или ошибка:", err);
            });
        });
    });
});

function filterVideos() {
    let filter = document.getElementById('search').value.toLowerCase();
    document.querySelectorAll('.video-item').forEach(item => {
        if(item.textContent.toLowerCase().includes(filter)) {
            item.style.display = "";
        } else {
            item.style.display = "none";
        }
    });
}
</script>
</body>
</html>`

func deletePack(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод (наш фронт шлёт DELETE)
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Достаем параметры из URL: ?mode=movie&name=Mr_Robot
	queryParams := r.URL.Query()
	mode := queryParams.Get("mode")
	name := queryParams.Get("name")

	// Базовая валидация, чтобы случайно не потереть лишнего
	if mode == "" || name == "" || strings.Contains(name, "..") || strings.Contains(name, "/") {
		http.Error(w, "Invalid parameters", http.StatusBadRequest)
		return
	}

	// Собираем прямой путь к папке контента
	targetPath := fmt.Sprintf("UserData/Films/%s", name)

	// Проверяем, существует ли вообще такая папка
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		http.Error(w, "Folder not found", http.StatusNotFound)
		return
	}

	// Удаляем папку со всем содержимым (медиа, картинки, meta.json)
	err := os.RemoveAll(targetPath)
	if err != nil {
		log.Printf("Ошибка удаления папки %s: %v", targetPath, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Успешно удален пакет: %s (тип: %s)", name, mode)

	// Отдаем фронту "ok", как он и ждет
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
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
	http.Handle("/films/", http.StripPrefix("/films/", http.FileServer(http.Dir("UserData/Films/"))))
	http.Handle("/s/", http.StripPrefix("/s/", http.FileServer(http.Dir("static"))))

	go cleanupQRSessions()
	log.Println("started")
	http.ListenAndServeTLS(":443", ".secure/certs/server.crt", ".secure/certs/server.key", nil)
}