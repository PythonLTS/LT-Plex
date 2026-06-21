document.addEventListener("DOMContentLoaded", () => {
    let currentStep = 1;
    const maxStep = 7;

    // Находим кнопку и окна один раз при загрузке страницы
    const nextButton = document.querySelector('.btn-next');
    const win1 = document.querySelector('.container-main');
    const win2 = document.querySelector('.container-languageSelect');

    // Главная функция переключения экранов
    function renderStep() {
        // Скрываем все окна по умолчанию (добавь сюда остальные окна, когда они появятся)
        if (win1) win1.style.display = "none";
        if (win2) win2.style.display = "none";

        // Показываем нужное окно в зависимости от текущего шага
        switch (currentStep) {
            case 1:
                if (win1) win1.style.display = "block";
                break;
            case 2:
                alert('Перешли на win2!!!');
                if (win2) win2.style.display = "block";
                break;
            case 3:
                alert("func work! (Шаг 3)");
                // Здесь будет логика для третьего окна
                break;
             case 4:

             case 5:

             case 6:

            default:
                alert("Конец формы или неизвестный шаг");
        }
    }

    // Вешаем ОДИН обработчик клика на кнопку
    if (nextButton) {
        nextButton.addEventListener('click', () => {
            if (currentStep < maxStep) {
                currentStep++; // Переходим на следующий шаг
                renderStep();  // Обновляем интерфейс
            } else {
                alert('Вы дошли до максимума!');
            }
        });
    }

    // Инициализация: показываем первый шаг при старте
    renderStep();
});