package main

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"
	"io"
	"strings"
	"net"
	"encoding/json"
)
type LTpack struct {
	Name 	string
	TypeContent string

}
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
	//Создает пакет ltpack и пихает в /Films
	//возвращая 1 если успех и 0 если нет
	return
}

func executeSettings(){
	//Приминение настроек (язык,тема,Доменное имя,автозапуск,включить ли дополнение)
	//если dns истина то записать в /etc/hosts ltplex.com и айпи
	//если автозапуск истина то делать systemd target или сервис
	//если включено дополнение , включить дополнение XD.
	//язык сохранить в settings.json как и тему а так же параметры что включено 
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

func storageUnpack(w http.ResponseWriter,r *http.Request){
	//принимает ltpack пакеты , нужно проверить сразу , тот ли это пакет, если да то проверить структуру если можно и распаковать в /Films(опять же я поменяю потом)
	return
}

func storageCapacity(w http.ResponseWriter,r *http.Request){
	//TEST только mb/gb
	w.Write([]byte("726GB"))
	//нужно вернуть число как в тест но настоящее свободное место на устройстве
	return
}

func storagestatus(w http.ResponseWriter,r *http.Request){
	//вернуть список /Films (путь сам поменяю), только типы и количество по типу (1 видео , 2 фильма, 20 сериалов,3 канала)
	return
}

func updateapp(w http.ResponseWriter, r *http.Request) {

	// ==========================================================================
	// ШАГ 1: ПОЛУЧЕНИЕ ИЗ ЛОКАЛЬНОГО ФАЙЛА ТЕКУЩЕЙ ВЕРСИИ ПРИЛОЖЕНИЯ
	// ==========================================================================

	// Читаем массив байт из файла, который лежит на один уровень выше текущей папки
	localBytes, err := os.ReadFile("../version")
	if err != nil {
		// Если файла нет или к нему нет доступа, отдаем клиенту ошибку 500 Internals Server Error
		http.Error(w, "Ошибка чтения локальной версии", http.StatusInternalServerError)
		return // Прерываем выполнение функции
	}

	// Переводим байты в строку и очищаем от случайных пробелов и переносов строк (\n, \r)
	currentVersion := strings.TrimSpace(string(localBytes))
	
	// Выводим в консоль сервера текущую версию для логирования
	fmt.Println("Текущая локальная версия:", currentVersion)


	// ==========================================================================
	// ШАГ 2: ЗАПРОС СВЕЖЕЙ ВЕРСИИ С REPO GITHUB (СЕТЕВОЙ ЗАПРОС)
	// ==========================================================================

	// Ссылка на raw-файл в репозитории GitHub, где всегда написана актуальная версия
	url := "https://raw.githubusercontent.com/PythonLTS/LT-Plex/main/version"

	// Создаем HTTP-клиент с жестким таймаутом в 5 секунд (чтобы сервер не завис, если гитхаб недоступен)
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Выполняем GET-запрос по указанному URL-адресу
	resp, err := client.Get(url)
	if err != nil {
		// Безопасный сценарий: если упал интернет или лежат сервера GitHub, 
		// возвращаем клиенту "0" (как будто обновлений нет), чтобы не ломать мастер установки
		w.Write([]byte("0"))
		return // Прерываем выполнение функции
	}
	
	// Обязательно закрываем тело ответа (Body) после завершения работы функции,
	// чтобы избежать утечки оперативной памяти и сетевых соединений
	defer resp.Body.Close()


	// ==========================================================================
	// ШАГ 3: ЧТЕНИЕ И ОБРАБОТКА ДАННЫХ ИЗ ОТВЕТА GITHUB
	// ==========================================================================

	// Выкачиваем всё содержимое ответа (тело страницы/файла) в виде массива байт
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// На случай, если соединение оборвалось во время скачивания данных
		w.Write([]byte("0"))
		return
	}

	// Переводим полученные байты с GitHub в строку и тоже очищаем от скрытых символов переноса строк
	latest := strings.TrimSpace(string(body))
	
	// Выводим в консоль сервера последнюю версию с гитхаба для логирования
	fmt.Println("Актуальная версия на GitHub:", latest)


	// ==========================================================================
	// ШАГ 4: СРАВНЕНИЕ ВЕРСИЙ И СВЕРКА РЕЗУЛЬТАТОВ
	// ==========================================================================

	// Если версия на гитхабе ОТЛИЧАЕТСЯ от нашей локальной версии
	if latest != currentVersion {
		
		// Отправляем HTTP статус 200 OK
		w.WriteHeader(http.StatusOK)
		
		// Пишем в ответ "1", сигнализируя веб-интерфейсу, что нужно показать плашку обновления
		w.Write([]byte("1")) 
		
		// ----------------------------------------------------------------------
		// ПЛАН НА БУДУЩЕЕ: Автоматическое фоновое обновление программы
		// ----------------------------------------------------------------------
		// Здесь можно будет запустить параллельный процесс (горутину) скачивания и установки:
		// go startSelfUpdate() 
		
	} else {
		
		// Если версии полностью совпадают (обновление не требуется)
		w.WriteHeader(http.StatusOK)
		
		// Пишем в ответ "0" — у пользователя установлена самая последняя версия
		w.Write([]byte("0")) 
    }
}

func root(w http.ResponseWriter,r *http.Request){
	http.ServeFile(w,r,"master_resources/Master.html")
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
	http.HandleFunc("/UploadPack", storageUnpack)
	http.HandleFunc("/Storage", storagestatus)
	http.HandleFunc("/StorageCapacity", storageCapacity)
	http.HandleFunc("/Update", updateapp)
	http.Handle("/resources/",http.StripPrefix("/resources/",http.FileServer(http.Dir("master_resources"))))
	http.HandleFunc("/",root)
	http.ListenAndServe(":8080",nil)
}
