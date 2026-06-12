document.addEventListener("DOMContentLoaded", function() {
  const back = document.getElementById("Back");
  const username = document.getElementById("user");
  const password = document.getElementById("pass");
  const sign = document.getElementById("Login");
  const notify = document.getElementById("Notification");
  const qrImg = document.querySelector("#qr-box img");

  // Переменные для контроля таймеров уведомлений
  let opacityTimeout;
  let displayTimeout;

  fetch("/profile-data", { credentials: "include" })
    .then(res => {
      if (res.ok) window.location.href = "profile";
    });

  function showNotification(message, bg = "rgba(0,0,0,0.15)", border = "rgb(0,0,0)") {
    // Сбрасываем старые таймеры анимации
    clearTimeout(opacityTimeout);
    clearTimeout(displayTimeout);

    notify.textContent = message;
    notify.style.cssText = `
      background-color: ${bg};
      border-color: ${border};
      display: flex;
      justify-content: center;
      align-items: center;
      user-select: none;
      opacity: 1;
      transition: opacity 0.9s;
      white-space: pre-line; /* Для поддержки переноса строк \n */
    `;

    opacityTimeout = setTimeout(() => (notify.style.opacity = "0"), 2500);
    displayTimeout = setTimeout(() => (notify.style.display = "none"), 3400);
  }

  back.addEventListener("click", () => {
    window.location.href = "/";
  });

  // Выносим логику авторизации в отдельную чистую функцию
  async function makeLogin() {
    const user = username.value.trim();
    const pass = password.value.trim();

    if (!user || !pass) {
      showNotification("Поля пусты", "rgba(0,0,0,0.1)", "white");
      return;
    }

    try {
      const response = await fetch("/api/loginData/", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ Username: user, Password: pass }),
        credentials: "include" 
      });

      if (response.ok) {
        showNotification("Успешно", "rgba(0,255,0,0.2)", "lime");
        setTimeout(() => (window.location.href = "/profile"), 1000);
      } else {
        const msg = await response.text();
        showNotification(msg, "rgba(0,0,0,0.1)", "white");
      }
    } catch (err) {
      showNotification("Ошибка сети", "rgba(0,0,0,0.1)", "white");
    }
  }

  // Привязываем функцию к клику по кнопке
  sign.addEventListener("click", makeLogin);

  // Слушаем нажатие Enter в инпутах и отправляем форму
  const handleEnter = (event) => {
    if (event.key === "Enter") {
      event.preventDefault();
      makeLogin();
    }
  };

  username.addEventListener("keydown", handleEnter);
  password.addEventListener("keydown", handleEnter);

  // Логика работы с QR-кодом
  async function fetchQRToken() {
    const res = await fetch("/qrdata/", { credentials: "include" });
    const blob = await res.blob();
    const token = res.headers.get("X-QR-Token"); 
    qrImg.src = URL.createObjectURL(blob);
    return token;
  }

  let qrToken = null;
  fetchQRToken().then(token => {
    qrToken = token;
    pollQRStatus();
  });

  async function pollQRStatus() {
    if (!qrToken) return;
    try {
      const res = await fetch("/data-status?token=" + encodeURIComponent(qrToken), {
        credentials: "include",
        headers: { "Accept": "application/json" }
      });

      if (!res.ok) {
        setTimeout(pollQRStatus, 2000);
        return;
      }

      const data = await res.json();

      if (data.status === "ok") {
        showNotification("QR успешно", "rgba(0,255,0,0.2)", "lime");
        setTimeout(() => (window.location.href = "/profile"), 700);
      } else {
        setTimeout(pollQRStatus, 2000);
      }
    } catch (err) {
      setTimeout(pollQRStatus, 5000);
    }
  }
});