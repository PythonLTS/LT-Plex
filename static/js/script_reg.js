document.addEventListener("DOMContentLoaded", function() {
  const userInput = document.getElementById("user");
  const passInput = document.getElementById("pass");
  const loginButton = document.getElementById("Login");
  const checkbox = document.getElementById("checkBox");
  const sign = document.getElementById("Sign-in");
  const notify = document.getElementById("Notification");
  
  // Переменные для хранения идентификаторов тайм-аутов
  let opacityTimeout;
  let displayTimeout;
  
  fetch("/profile-data", { credentials: "include" })
    .then(res => {
      if (res.ok) window.location.href = "profile";
    });

  function showNotification(message, bg = "rgba(0,0,0,0.15)", border = "rgb(0,0,0)") {
    clearTimeout(opacityTimeout);
    clearTimeout(displayTimeout);

    notify.textContent = message;
    
    // Меняем только то, что нужно динамически
    notify.style.backgroundColor = bg;
    notify.style.borderColor = border;
    notify.style.display = "flex";
    notify.style.justifyContent = "center";
    notify.style.alignItems = "center";
    notify.style.userSelect = "none";
    notify.style.opacity = "1";
    notify.style.transition = "opacity 1s";
    notify.style.whiteSpace = "pre-line";
    
    opacityTimeout = setTimeout(() => notify.style.opacity = "0", 2500);
    displayTimeout = setTimeout(() => notify.style.display = "none", 3400);
  }

  sign.addEventListener("click", () => {
    if (!checkbox.checked){
      showNotification("Примите Соглашение", "rgba(0,0,0,0.15)", "rgb(255,255,255)");
      return;
    }
    window.location.href = "/sign";
  });

  // 1. Выносим всю логику в отдельную функцию
function makeLogin() {
    const username = userInput.value.trim();
    const password = passInput.value.trim();

    const validations = [
      { check: !checkbox.checked, message: "Примите Соглашение" },
      { check: username.length === 0, message: "Поля пусты" },
      { check: password.length === 0, message: "Придумайте пароль" },
      { check: username.length > 20, message: "Максимум 20 символов" },
      { check: password.length < 8, message: "Пароль: минимум 8 символов" },
      { check: password.length > 15, message: "Пароль: максимум 15 символов" }
    ];

    for (let rule of validations) {
      if (rule.check) {
        showNotification(rule.message, "rgba(0,0,0,0.15)", "rgb(255,255,255)");
        return;
      }
    }

    fetch("/api/registerData/", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ Username: username, Password: password })
    })
    .then(res => {
      if (!res.ok) throw new Error();
      return res.text();
    })
    .then(statusText => {
      if (statusText === "exists") {
        showNotification("Имя уже занято", "rgba(0,0,0,0.15)", "rgb(255,0,0)");
        return;
      }

      showNotification("Успешно!", "rgba(0,128,0,0.15)", "rgb(255,255,255)");
      setTimeout(() => window.location.href = "/profile", 2000);
    })
    .catch(() => {
      showNotification("Что-то не так\nпопробуйте снова", "rgba(0,0,0,0.15)", "rgb(255,0,0)");
    });
  }

  // 2. Вешаем эту функцию на клик по кнопке
  loginButton.addEventListener("click", makeLogin);

  // 3. Вешаем её же на нажатие Enter в инпутах
  const handleEnter = (event) => {
    if (event.key === "Enter") {
      event.preventDefault();
      makeLogin(); // Вызываем логику напрямую
    }
  };

  userInput.addEventListener("keydown", handleEnter);
  passInput.addEventListener("keydown", handleEnter);
});