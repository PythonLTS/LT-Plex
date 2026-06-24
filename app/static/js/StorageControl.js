// Ждем полной загрузки DOM дерева
document.addEventListener("DOMContentLoaded", () => {
    
    // 1. Автоматический запуск сканирования при заходе на страницу
    scanStorage();

    // 2. Находим кнопки в DOM
    const btnCreate = document.getElementById("btn-create");
    const btnUpload = document.getElementById("btn-upload");
    const btnScan = document.getElementById("btn-scan");

    // 3. Вешаем обработчики событий (Click)
    if (btnCreate) {
        btnCreate.addEventListener("click", createLTpack);
    }

    if (btnUpload) {
        btnUpload.addEventListener("click", uploadLTpack);
    }

    if (btnScan) {
        btnScan.addEventListener("click", scanStorage);
    }
});

// --- Описание функций ---

function scanStorage() {
    console.log("Вызвана функция: scanStorage (Сканирование хранилища...)");
    // Твой код для сканирования
}

function createLTpack() {
    console.log("Вызвана функция: createLTpack (Создание пакета...)");
    // Твой код для создания пакета
}

function uploadLTpack() {
    console.log("Вызвана функция: uploadLTpack (Загрузка пакета...)");
    // Твой код для загрузки
}