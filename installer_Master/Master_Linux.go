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

func executeSettings(language string,smart bool,dns bool,autorun bool,theme string) (bool,error){
	path := "master_resources/master_data.json"
	var name string
	if (dns){
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			fmt.Printf("[Ошибка!] не удалось получить интерфейсы: %v", err)
			os.Exit(1)
		}

		for _, address := range addrs {
			if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					myIP := ipnet.IP.String()

					name = "\n" + myIP + " ltplex.com\n"
					log.Println("Доменное имя Включено :",name)

				}
			}
		}
	}
	log.Println("Язык сохранен: ",language)
	if (smart){
		log.Println("SmartTV Дополнение Сохранено: ",smart)
	}
	if (autorun){
		log.Println("Автозапуск включен!")
	}
	log.Println("Сохранена тема: ",theme)
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
		log.Println("[Error] Не удалось создать JSON:", err)
		return false,err
	}

	err = os.WriteFile(path, jsonData, 0644)
	if err != nil {
		log.Println("[Error] Не удалось сохранить настройки:", err)
		return false,err
	}

	log.Println("[Debug] Настройки сохранены и пременены Успешно !!!!!!")

	//Приминение настроек (язык,тема,Доменное имя,автозапуск,включить ли дополнение)
	//если dns истина то записать в /etc/hosts ltplex.com и айпи
	//если автозапуск истина то делать systemd target или сервис
	//если включено дополнение ,включить его
	//язык сохранить в settings.json в самом приложении как и тему а так же параметры что включено 
	return true,nil
}

func savesettings(w http.ResponseWriter,r *http.Request){
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
	
	fmt.Printf("[Debug] Настройки Пришли!\n:\nФлаг запуска:%t\nЯзык:%s\nТема:%s\nДополнение:%t\nДоменное имя:%t\nАвтозапуск:%t\n",data.Status,data.Language,data.Theme,data.SmartPlex,data.Dns,data.Autostart)
	flag,err := executeSettings(data.Language,data.SmartPlex,data.Dns,data.Autostart,data.Theme)
	if err != nil {
		log.Println("[Error]", err)
		http.Error(w, "internal error", 500)
		return
	}
	if flag {
		log.Println("Подтверждение успешного сохранения!")

		return

	}
	log.Println("[Error] что-то пошло не так: нет Подтверждения")
	w.WriteHeader(http.StatusOK)
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
func startInstaller(){

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
