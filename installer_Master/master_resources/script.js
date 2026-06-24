document.addEventListener("DOMContentLoaded", () => {
    const status = true;
    let theme = "dark";
    let language = "ru";
    let updateInfo = null;
    let dns = false;
    let smartPlex = false;
    let autostart = false;
    let updateInterval = null;
    let updateTimeout = null;

    const UPDATE_TIMEOUT = 10000; // 30 секунд
    const STEPS = ["welcome", "lang", "update", "features", "finish"];

    // ===== THEME =====
    function toggleTheme(){
      theme = (theme === "dark") ? "light" : "dark";
      document.body.className = theme;
    }


    function startWaitingUpdate(){
      const loader = document.getElementById("loader");
      const updateBtns = document.getElementById("updateBtns");
      const updateText = document.getElementById("updateText");

      loader.classList.remove("hidden");

      // таймаут (защита от зависания)
      updateTimeout = setTimeout(() => {
        stopWaitingUpdate();

        loader.classList.add("hidden");
        updateBtns.classList.remove("hidden");

        updateText.innerText = "Ошибка обновления";
      }, UPDATE_TIMEOUT);

      // polling
      updateInterval = setInterval(async () => {
        try {
          const res = await fetch("/UpdateStatus");
          const data = await res.json();

          if (data.status === "SuccessUpdate") {
              stopWaitingUpdate();

              const loader = document.getElementById("loader");
              const success = document.getElementById("success");

              loader.classList.add("hidden");
              success.classList.remove("hidden");

              setTimeout(() => {
                next("features");
              }, 2000);
            }

        } catch (e) {
          console.log("waiting update...");
        }
      }, 1500);
    }

    function stopWaitingUpdate(){
      if (updateInterval) {
        clearInterval(updateInterval);
        updateInterval = null;
      }

      if (updateTimeout) {
        clearTimeout(updateTimeout);
        updateTimeout = null;
      }
    }

    // ===== HASH ROUTER =====
    window.addEventListener("hashchange", router);

    function router(){
      const h = location.hash.replace("#","");
      showStep(h);
    }

    // ===== NAV =====
    function next(step){
      location.hash = step;
    }

    function prev() {
      const currentStep = location.hash.replace("#", "") || STEPS[0];
      const currentIndex = STEPS.indexOf(currentStep);

      if (currentIndex > 0) {
        location.hash = STEPS[currentIndex - 1];
      }
    }

    function showStep(step){
      document.querySelectorAll(".card").forEach(c => c.classList.add("hidden"));

      const map = {
        welcome: "step-welcome",
        lang: "step-lang",
        update: "step-update",
        features: "step-features",
        finish: "step-finish"
      };

      const activeStep = map[step] || "step-welcome";
      document.getElementById(activeStep).classList.remove("hidden");
      
      updateDots(map[step] ? step : STEPS[0]);

      if(step === "update") checkUpdate();
      if(step === "finish") finish();
    }

    // ===== DOTS =====
    function updateDots(step){
      let index = STEPS.indexOf(step);

      for(let i = 1; i <= 5; i++){
        const dot = document.getElementById("d" + i);
        if (dot) dot.classList.remove("active");
      }

      for(let i = 1; i <= index + 1; i++){
        const dot = document.getElementById("d" + i);
        if (dot) dot.classList.add("active");
      }
    }

    // ===== LANGUAGE =====
    function setLang(l){
      language = l;
      next("update");
    }

    // ===== UPDATE CHECK =====
    async function checkUpdate(){
      const updateText = document.getElementById("updateText");
      const updateBtns = document.getElementById("updateBtns");
      
      const btnRetry = document.getElementById("btn-retry");
      const btnApply = document.getElementById("btn-apply");
      const btnSkip = document.getElementById("btn-skip");

      // На время проверки скрываем весь блок кнопок и ставим текст
      updateBtns.classList.add("hidden");
      updateText.innerText = "Проверка обновлений...";

      try {
        const res = await fetch("/CheckUpdate");
        const data = await res.json();

        updateInfo = data;

        if (data.update === true) {
          updateText.innerText = `Обновление доступно ${data.from}\nТекущая версия ${data.to}`;
          
          // ЕСЛИ ОБНОВЛЕНИЕ ЕСТЬ: показываем "Обновить" и "Продолжить", скрываем "Повторить"
          btnApply.classList.remove("hidden");
          btnSkip.classList.remove("hidden");
          btnRetry.classList.add("hidden");
          
          // Текст кнопки продолжения можно сделать более контекстным
          btnSkip.innerText = "Продолжить без обновления";
        } else {
          updateText.innerText = `У вас актуальная версия: ${data.version}`;
          
          // ЕСЛИ ОБНОВЛЕНИЯ НЕТ: показываем "Повторить" и "Продолжить", скрываем "Обновить"
          btnApply.classList.add("hidden");
          btnSkip.classList.remove("hidden");
          btnRetry.classList.remove("hidden");
          
          btnSkip.innerText = "Продолжить";
        }

      } catch(e) {
        updateText.innerText = "Ошибка проверки";
        
        // ПРИ ОШИБКЕ: показываем "Повторить" и "Продолжить", скрываем "Обновить"
        btnApply.classList.add("hidden");
        btnSkip.classList.remove("hidden");
        btnRetry.classList.remove("hidden");
        
        btnSkip.innerText = "Продолжить";
      }

      // Показываем сформированный блок кнопок
      updateBtns.classList.remove("hidden");
    }

    // ===== APPLY UPDATE =====
    async function applyUpdate(){
      if(!updateInfo) return;

      const res = await fetch("/GetUpdate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          update: `${updateInfo.from}->${updateInfo.to}`
        })
      });

      if (res.status === 200) {
        document.getElementById("updateBtns").classList.add("hidden");
        startWaitingUpdate();
      }
    }

    // ===== FINISH =====
    async function finish(){
        dns = document.getElementById("dns").checked,
        smartPlex = document.getElementById("smart").checked,
        autostart = document.getElementById("autostart").checked

      await fetch("/saveSettings", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          status,
          theme,
          language,
          dns,
          smartPlex,
          autostart
        })
      });

      window.close();
    }

    // ===== ЭКСПОРТ ДЛЯ HTML (Важное исправление) =====
    // Привязываем функции к window, чтобы их продолжали видеть инлайновые атрибуты onclick
    window.next = next;
    window.prev = prev;
    window.toggleTheme = toggleTheme;
    window.setLang = setLang;
    window.checkUpdate = checkUpdate;
    window.applyUpdate = applyUpdate;
    window.finish = finish;

    // ===== INIT =====
    document.body.className = theme;
    if(!location.hash) location.hash = "welcome";
    router();
});