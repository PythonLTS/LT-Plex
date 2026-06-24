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

					name = "\n" + myIP + ":8080" + " ltplex.com\n"
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

	log.Println("[Debug] Настройки сохранены и пременены Успешно!!!!!!")

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
	defer w.WriteHeader(http.StatusOK)
	defer r.Body.Close()
	
	fmt.Printf("[Debug] Настройки Пришли!\n:\nФлаг запуска:%t\nЯзык:%s\nТема:%s\nДополнение:%t\nДоменное имя:%t\nАвтозапуск:%t\n",data.Status,data.Language,data.Theme,data.SmartPlex,data.Dns,data.Autostart)
	flag,err := executeSettings(data.Language,data.SmartPlex,data.Dns,data.Autostart,data.Theme)
	if err != nil {
		log.Fatal("[Error] saving Data | ",err)
	}
	if flag {
		log.Println("Подтверждение успешного сохранения!")

		return

	}
	log.Println("[Error] что-то пошло не так: нет Подтверждения")
}



func updateApp(w http.ResponseWriter, r *http.Request) {
	log.Println("[Debug] /updateApp Обновление... ")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(""))
}

func checkUpdate(w http.ResponseWriter, r *http.Request) {
	log.Println("[Debug] /checkUpdate Проверка обновлений...")
	data, err := os.ReadFile("../version")
	if err != nil {
		fmt.Println("error")
		panic(err)
	}
	fmt.Println(string(data))

	w.Header().Set("Content-Type", "application/json")

	// Логика: проверяем, нужно ли обновление (здесь для примера хардкод)
	hasUpdate := true

	var response UpdateResponse

	if hasUpdate {
		// Если обновление есть, отдаем true, старую и новую версию
		response = UpdateResponse{
			Update: true,
			From:   "x.x.x",
			To:     string(data),
		}
	} else {
		// Если обновления нет, отдаем false и текущую актуальную версию
		response = UpdateResponse{
			Update:  false,
			Version: "x.x.x",
		}
	}
	// Упаковываем структуру в JSON и отправляем в ответ
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
}
func checkUpdateStatus(w http.ResponseWriter,r *http.Request){
	response := map[string]string{
		"status": "SuccessUpdate",
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
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
	http.HandleFunc("/GetUpdate", updateApp)
	http.HandleFunc("/CheckUpdate", checkUpdate)
	http.HandleFunc("/UpdateStatus", checkUpdateStatus)
	http.Handle("/resources/",http.StripPrefix("/resources/",http.FileServer(http.Dir("master_resources"))))
	http.HandleFunc("/",root)
	http.ListenAndServe(":8080",nil)
}
