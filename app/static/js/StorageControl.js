
// Объявление функций, которые ты просил связать
async function scanStorage() {
    console.log("Вызвана функция: scanStorage()");
    try {
    const response = await fetch('/api/StorageInfo?mode=list');
    
    // Проверяем, что запрос прошёл успешно (статус 200-299)
    if (!response.ok) {
      throw new Error(`Ошибка HTTP: ${response.status}`);
    }

    const data = await response.json();
    
    // Здесь data будет именно в том формате, который ты ожидал:
    // { films: xx, channels: xx, series: x, freeSpace: "xxx" }
    console.log(data.series);
    
  } catch (error) {
    console.error('Не удалось получить данные о хранилище:', error);
    // Можешь вернуть дефолтные значения на случай ошибки, чтобы код не падал
    return null; 
  }
}
    // Сюда пойдет логика сканирования хранилища

function createLTpack() {
    console.log("Вызвана функция: createLTpack()");
    // Сюда пойдет логика создания пакета
}

function uploadLTpack() {
    console.log("Вызвана функция: uploadLTpack()");
    // Сюда пойдет логика загрузки пакета
}

// Ждем полной загрузки DOM дерева
document.addEventListener("DOMContentLoaded", () => {
    
    // 1. Автоматический запуск сканирования при заходе на страницу
    scanStorage();

    // 2. Привязка обработчиков клика к кнопкам по их ID
    document.getElementById("btnCreate").addEventListener("click", createLTpack);
    document.getElementById("btnUpload").addEventListener("click", uploadLTpack);
    document.getElementById("btnScan").addEventListener("click", scanStorage);
});
