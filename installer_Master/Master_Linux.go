package main

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"
	"net"
	"encoding/json"
	"log"
	"strings"
	"io"
	"os/exec"
	"path/filepath"
)
type Settings struct {
	  Status 	bool   `json:"status"`
	  Language  string `json:"language"`
	  Theme 	string `json:"theme"`
	  SmartPlex bool   `json:"smartPlex"`
	  Dns		bool   `json:"dns"`
	  Autostart bool	`json:"autostart"`
}
type UpdateResponse struct {
	Update  bool   `json:"update"`  
	From    string `json:"from"`    
	To      string `json:"to"`      
	Version string `json:"version"` 
}

var downloadStatus string = "none"


func CopyDir(src string, dst string) error {
	// 1. Получаем информацию об исходной папке
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("исходная папка не найдена: %w", err)
	}

	// 2. Создаем целевую папку с теми же правами доступа
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("не удалось создать целевую папку: %w", err)
	}

	// 3. Обходим все файлы и подпапки внутри src
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Вычисляем относительный путь, чтобы воссоздать структуру в dst
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, relPath)

		// Если это папка — создаем её в целевой директории
		if d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(targetPath, info.Mode())
		}

		// Если это файл — копируем его содержимое
		return copyFile(path, targetPath)
	})
}

// Вспомогательная функция для копирования одного файла
func copyFile(srcFile, dstFile string) error {
	out, err := os.Create(dstFile)
	if err != nil {
		return err
	}
	defer out.Close()

	in, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer in.Close()

	// Переносим права доступа файла (например, исполняемый ли он)
	srcInfo, err := in.Stat()
	if err == nil {
		_ = os.Chmod(dstFile, srcInfo.Mode())
	}

	// Эффективно стримим данные из одного файла в другой
	_, err = io.Copy(out, in)
	return err
}

func getLocalIp() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Printf("[Ошибка!] не удалось получить интерфейсы: %v", err)
		os.Exit(1)
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				myIP := ipnet.IP.String()
				return "http://"+myIP+":8080"
			}
		}
	}
	return "http://localhost:8080"
}

func verifyOSName(){
	fmt.Println("# # Проверка ОС # #")
	if (runtime.GOOS == "linux") {
		time.Sleep(500 * time.Millisecond)
		fmt.Println("ОС: Linux")

	} else if (runtime.GOOS == "windows"){
		fmt.Println("Это не Linux!\nВыберите Master_Linux!\nВыход ...")
		os.Exit(1)

	}
	return
}

func executeSettings(language string, smart bool, dns bool, autorun bool, theme string) (bool, error) {
	// Временные и постоянные пути
	tmpExtractDir := "../app"
	targetAppDir := "/opt/ltplex/app"
	log.Println("execute settings ....")
	// Шаг 1: Сначала запускаем установку/перенос структуры файлов
	err := MainInstaller(tmpExtractDir, targetAppDir)
	if err != nil {
		return false, fmt.Errorf("критическая ошибка установки: %w", err)
	}
	log.Println("execute settings ....2")
	// Шаг 2: Применение настроек DNS (Запись в /etc/hosts)
	if dns {
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			log.Printf("[Ошибка] не удалось получить интерфейсы: %v", err)
		} else {
			for _, address := range addrs {
				if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						myIP := ipnet.IP.String()
						log.Printf("Доменное имя включено для IP: %s -> ltplex.com\n", myIP)
						
						// Логика записи в /etc/hosts
						hostsLine := fmt.Sprintf("\n%s ltplex.com\n", myIP)
						f, err := os.OpenFile("/etc/hosts", os.O_APPEND|os.O_WRONLY, 0644)
						if err == nil {
							_, _ = f.WriteString(hostsLine)
							f.Close()
						} else {
							log.Println("[Error] Нет прав на запись в /etc/hosts (нужен sudo/root):", err)
						}
						break // Нашли первый IPv4 и хватит
					}
				}
			}
		}
	}
	log.Println("execute settings ....3")
	// Шаг 3: Автозапуск через Systemd
	if autorun {
		log.Println("Настройка автозапуска systemd...")
		serviceConfig := `[Unit]
Description=LT-Plex App
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/ltplex/app
ExecStart=/opt/ltplex/app/server
Restart=always

[Install]
WantedBy=multi-user.target
`
		servicePath := "/etc/systemd/system/ltplex.service"
		err = os.WriteFile(servicePath, []byte(serviceConfig), 0644)
		if err == nil {
			// Перезапускаем демона и включаем сервис в автозапуск
			_, _ = exec.Command("systemctl", "daemon-reload").Output()
			_, _ = exec.Command("systemctl", "enable", "ltplex.service").Output()
			_, _ = exec.Command("systemctl", "start", "ltplex.service").Output()
			log.Println("Автозапуск успешно настроен в systemd!")
		} else {
			log.Println("[Error] Не удалось создать systemd сервис (нужен root):", err)
		}
	}

	log.Printf("Выдача прав setcap для бинарника: /opt/ltplex/app/server\n")
	
	cmd := exec.Command("setcap", "cap_net_bind_service=+ep", "/opt/ltplex/app/server")
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("setcap [Debug] ",output)
		return false, err
	}
	log.Println("Успешно ",output)
	log.Println("Права на 443 порт успешно выданы!")

	appSettingsPath := filepath.Join(targetAppDir, "master_resources/master_data.json")
	_ = os.MkdirAll(filepath.Dir(appSettingsPath), 0755)

	settings := Settings{
		Status:    true,
		Language:  language,
		Theme:     theme,
		SmartPlex: smart,
		Dns:       dns,
		Autostart: autorun,
	}

	jsonData, err := json.MarshalIndent(settings, "", "    ")
	if err != nil {
		return false, fmt.Errorf("не удалось создать JSON: %w", err)
	}

	err = os.WriteFile(appSettingsPath, jsonData, 0644)
	if err != nil {
		return false, fmt.Errorf("не удалось сохранить настройки в приложение: %w", err)
	}

	// === НАШЕ НОВОЕ ДОПОЛНЕНИЕ: Копируем master_resources/master_data.json в app/UserData/settings.json ===
	userSettingsPath := filepath.Join(targetAppDir, "UserData/settings.json")
	
	// На всякий случай создаем папку UserData внутри установленного приложения, если её нет
	if err := os.MkdirAll(filepath.Dir(userSettingsPath), 0755); err != nil {
		log.Println("[Error] Не удалось создать папку UserData:", err)
	} else {
		// Копируем файл (вызываем нашу функцию copyFile, которую мы писали для CopyDir)
		if err := copyFile(appSettingsPath, userSettingsPath); err != nil {
			log.Println("[Error] Не удалось скопировать настройки в UserData/settings.json:", err)
		} else {
			log.Println("[Debug] Конфиг успешно продублирован в UserData/settings.json")
		}
	}

	log.Println("[Debug] Настройки сохранены и применены успешно!")
	return true, nil
}

func savesettings(w http.ResponseWriter, r *http.Request) {
	log.Println("[Debug] Сохранение настроек")
	var data Settings

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&data)
	if err != nil {
		log.Println("[Error] ошибка savesettings:", err)
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	fmt.Printf("[Debug] Настройки Пришли! Язык:%s, Тема:%s\n", data.Language, data.Theme)
	
	flag, err := executeSettings(data.Language, data.SmartPlex, data.Dns, data.Autostart, data.Theme)
	if err != nil {
		log.Println("[Error]1", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if flag {
		log.Println("Подтверждение успешного сохранения!")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success"}`))
		return
	}

	log.Println("[Error] что-то пошло не так: нет Подтверждения")
	http.Error(w, "unknown error", http.StatusInternalServerError)
}

func startUpdater(){
	log.Println("[Debug] запуск updater...")

	cmd := exec.Command("./updater")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		log.Println("[Error] не удалось запустить updater:", err)
		return
	}

	log.Println("[Debug] updater запущен")
}

func MainInstaller(extractedTargetDir string, finalAppDir string) error {
	if _, err := os.Stat(extractedTargetDir); os.IsNotExist(err) {
		return fmt.Errorf("ошибка установки: папка с обновлением %s не найдена", extractedTargetDir)
	}

	if _, err := os.Stat(finalAppDir); err == nil {
		fmt.Printf("Удаляем старую версию приложения из %s...\n", finalAppDir)
		if err := os.RemoveAll(finalAppDir); err != nil {
			return fmt.Errorf("не удалось удалить старую версию: %w", err)
		}
	}

	fmt.Printf("Копируем новые файлы из %s в %s...\n", extractedTargetDir, finalAppDir)
	// Внимание: убрали os. перед CopyDir, вызываем нашу кастомную функцию
	if err := CopyDir(extractedTargetDir, finalAppDir); err != nil {
		return fmt.Errorf("ошибка при копировании файлов приложения: %w", err)
	}

	return nil
}

func GetAppUpdate(w http.ResponseWriter, r *http.Request) {
	log.Println("[Debug] /updateApp Обновление... ")
	downloadStatus = "downloading"
	go func (){
		url := "https://github.com/PythonLTS/LT-Plex/archive/refs/heads/main.zip"

		resp, err := http.Get(url)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		file, err := os.Create("update.zip")
		if err != nil {
			panic(err)
		}
		defer file.Close()

		io.Copy(file, resp.Body)
		downloadStatus = "done"
		startUpdater()
	}()
	json.NewEncoder(w).Encode(map[string]string{
		"status": downloadStatus,
	})
}


//Проверка Обновлений \ Check Updates
func checkUpdate(w http.ResponseWriter, r *http.Request) {
	log.Println("[Debug] /checkUpdate Проверка обновлений...")

	
	localBytes, err := os.ReadFile("../version")
	if err != nil {
		http.Error(w, "failed to read local version", http.StatusInternalServerError)
		return
	}
	localVersion := strings.TrimSpace(string(localBytes))

	
	resp, err := http.Get("https://raw.githubusercontent.com/PythonLTS/LT-Plex/refs/heads/main/version")
	if err != nil {
		http.Error(w, "failed to fetch remote version", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	remoteBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "failed to read remote version", http.StatusInternalServerError)
		return
	}
	remoteVersion := strings.TrimSpace(string(remoteBytes))

	
	hasUpdate := remoteVersion != localVersion

	w.Header().Set("Content-Type", "application/json")

	response := UpdateResponse{
		Update:  hasUpdate,
		From:    localVersion,
		To:      remoteVersion,
		Version: localVersion,
	}

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
func checkUpdateStatus(w http.ResponseWriter,r *http.Request){
	json.NewEncoder(w).Encode(map[string]string{
		"status": downloadStatus,
	})
}

func root(w http.ResponseWriter,r *http.Request){
	http.ServeFile(w,r,"master_resources/Master.html")
	return
}

func main(){
	fmt.Println("### Инициализация ###")
	if os.Geteuid() != 0 {
	    log.Fatal("Запустите программу через sudo")
	    os.Exit(1)
	}
	verifyOSName()
	fmt.Println("Проверка прошла успешно!")
	time.Sleep(500 * time.Millisecond)
	fmt.Println("### Запуск установщика ###")
	time.Sleep(500 * time.Millisecond)
	fmt.Println("Успешно!\nОткройте в браузере сайт :\n",getLocalIp())
	http.Handle("/sources/",http.StripPrefix("/sources/",http.FileServer(http.Dir("master_resources/"))))
	http.HandleFunc("/saveSettings", savesettings)
	http.HandleFunc("/GetUpdate", GetAppUpdate)
	http.HandleFunc("/CheckUpdate", checkUpdate)
	http.HandleFunc("/UpdateStatus", checkUpdateStatus)
	http.Handle("/resources/",http.StripPrefix("/resources/",http.FileServer(http.Dir("master_resources"))))
	http.HandleFunc("/",root)
	http.ListenAndServe(":8080",nil)
}
