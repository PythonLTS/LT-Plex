document.addEventListener("DOMContentLoaded", async function () {

    document.getElementById("back").addEventListener("click", () => {
        window.location.href = "/profile";
    });

    // 1. Получаем данные профиля (избранное) и список фильмов параллельно
    let userFavorites = [];
    let filmsList = [];

    try {
        const [profileRes, filmsRes] = await Promise.all([
            fetch("/profile-data", { credentials: "include" }),
            fetch("/films", { credentials: "include" })
        ]);

        if (!profileRes.ok || !filmsRes.ok) throw new Error("Unauthorized or Data error");

        const profileData = await profileRes.json();
        userFavorites = profileData.favorites || [];
        
        const fetchedFilms = await filmsRes.json();
        // ЗАЩИТА: Если бэк всё-таки прислал null, превращаем его в пустой массив
        filmsList = fetchedFilms || []; 
    } catch (e) {
        console.error("Ошибка загрузки данных:", e);
        return;
    }

    const listContainer = document.getElementById("list");
    const mq = window.matchMedia("(max-width: 600px)");

    // Создаем счетчик контента динамически
    const counterDisplay = document.createElement("div");
    counterDisplay.id = "film-counter";
    document.body.insertBefore(counterDisplay, listContainer);

    // 2. Функция для генерации карточек фильмов на лету
    function renderFilms(films) {
        listContainer.innerHTML = ""; // Очищаем контейнер

        // Считаем количество элементов (с защитой от null/undefined)
        const count = (films && Array.isArray(films)) ? films.length : 0;
        counterDisplay.textContent = `Всего Архивов: ${count}`;

        if (count === 0) {
            listContainer.innerHTML = "<p style='color:gray; padding:20px; font-size:18px;text-align:center;'>Хранилище пусто <br> Вы можете загрузить сюда ваш Контент через Админ Панель<br>Подробнее в README.md</p>";
            return;
        }

        films.forEach(film => {
            const block = document.createElement("div");
            block.id = film.id;
            block.className = "film-card"; // Можешь стилизовать в CSS

            // Структурируем внутренности карточки
            block.innerHTML = `
                <img src="${film.logo}" alt="${film.name}">
                <h3>${film.name}</h3>
                <button class="watch">Смотреть</button>
                <button class="favorite"></button>
            `;

            const favBtn = block.querySelector(".favorite");
            const watchBtn = block.querySelector(".watch");

            // Переход к просмотру (учитывая твою структуру папок)
            watchBtn.addEventListener("click", () => {
                window.location.href = `/films/${film.id}/${film.id}.html`;
            });

            // Логика кнопки Избранного
            const updateFavButton = () => {
                if (userFavorites.includes(film.id)) {
                    favBtn.textContent = "Удалить из избранного";
                    favBtn.dataset.state = "added";
                } else {
                    favBtn.textContent = "В избранное";
                    favBtn.dataset.state = "not_added";
                }
            };
            updateFavButton();

            favBtn.addEventListener("click", async () => {
                const isAdded = favBtn.dataset.state === "added";
                const url = isAdded ? "/rmfv" : "/addfv";

                try {
                    const res = await fetch(url, {
                        method: "POST",
                        credentials: "include",
                        headers: { "Content-Type": "application/json" },
                        body: JSON.stringify({ movie: film.id })
                    });

                    if (res.ok) {
                        if (isAdded) {
                            const i = userFavorites.indexOf(film.id);
                            if (i !== -1) userFavorites.splice(i, 1);
                        } else {
                            userFavorites.push(film.id);
                        }
                        updateFavButton();
                    }
                } catch (err) {
                    console.error("Ошибка работы с избранным:", err);
                }
            });

            listContainer.appendChild(block);
        });
    }

    // 3. Создаем инпут поиска динамически
    const searchInput = document.createElement("input");
    searchInput.placeholder = "Поиск фильмов...";
    searchInput.id = "search";

    function applySearchStyles(isMobile) {
        const styles = isMobile ? {
            position: "relative", left: "6%", top: "160px", width: "340px", height: "50px",
            fontSize: "20px", borderRadius: "6px", border: "none", paddingLeft: "8px",
            outline: "none", background: "#f5f5f5"
        } : {
            position: "relative", left: "35px", top: "7px", margin: "10px", width: "260px",
            height: "32px", fontSize: "18px", borderRadius: "6px", border: "none",
            paddingLeft: "8px", outline: "none", background: "#f5f5f5"
        };
        Object.assign(searchInput.style, styles);
    }
    mq.addEventListener("change", (e) => applySearchStyles(e.matches));
    applySearchStyles(mq.matches);
    document.body.insertBefore(searchInput, listContainer);

    // 4. Логика фильтрации (только по поисковой строке)
    function applyFilters() {
        const query = searchInput.value.toLowerCase().trim();

        const filteredFilms = filmsList.filter(film => {
            return film.id.replace(/_/g, " ").toLowerCase().includes(query) || 
                   film.name.toLowerCase().includes(query);
        });

        renderFilms(filteredFilms);
    }

    searchInput.addEventListener("input", applyFilters);

    // Первый рендер всех фильмов при загрузке
    renderFilms(filmsList);
});