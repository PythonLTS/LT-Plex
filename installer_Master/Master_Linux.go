package main

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"
	"net"
	"encoding/json"
)

type Settings struct {
	  Status 	bool   `json:"status"`
	  Language  string `json:"language"`
	  Theme 	string `json:"theme"`
	  SmartPlex bool   `json:"smartPlex"`
	  Dns		bool   `json:"dns"`
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

func makeLTpack(){
	return
}

func executeSettings(){
	return
}

func savesettings(w http.ResponseWriter,r *http.Request){

	var data Settings

	decoder := json.NewDecoder(r.Body)

	
	err := decoder.Decode(&data)
	if err != nil {
		fmt.Println("ошибка savesettings")
		return
	}
	fmt.Printf("Настройки Пришли!\n:\nФлаг запуска:%t\nЯзык:%s\nТема:%s\nДополнение:%t\nДоменное имя:%t\n",data.Status,data.Language,data.Theme,data.SmartPlex,data.Dns)
	w.WriteHeader(http.StatusOK)
}

func storage(w http.ResponseWriter,r *http.Request){
	return
}

func storagecapacity(w http.ResponseWriter,r *http.Request){
	w.Write([]byte("100"))
	return
}

func storagestatus(w http.ResponseWriter,r *http.Request){
	return
}

func updateapp(w http.ResponseWriter,r *http.Request){
	return
}

func root(w http.ResponseWriter,r *http.Request){
	http.ServeFile(w,r,"master_resources/Welcome.html")
	return
}

func main(){
	fmt.Println("### Инициализация ###")
	verifyOSName()
	fmt.Println("Проверка прошла успешно!")
	time.Sleep(500 * time.Millisecond)
	fmt.Println("### Запуск установщика ###")
	time.Sleep(500 * time.Millisecond)
	fmt.Println("Успешно!\nОткройте в браузере сайт :\n",getLocalIp())
	http.Handle("/sources/",http.StripPrefix("/sources/",http.FileServer(http.Dir("master_resources/"))))
	http.HandleFunc("/saveSettings", savesettings)
	http.HandleFunc("/UploadPack", storage)
	http.HandleFunc("/Storage", storagestatus)
	http.HandleFunc("/StorageCapacity", storagecapacity)
	http.HandleFunc("/Update", updateapp)
	http.HandleFunc("/",root)
	http.ListenAndServe(":8080",nil)
}
