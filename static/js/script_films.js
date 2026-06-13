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
        filmsList = await filmsRes.json(); // Получаем массив {id, name, logo} с сервера
    } catch (e) {
        console.error("Ошибка загрузки данных:", e);
        return;
    }

    const listContainer = document.getElementById("list");
    const mq = window.matchMedia("(max-width: 600px)");

    // 2. Функция для генерации карточек фильмов на лету
    function renderFilms(films) {
        listContainer.innerHTML = ""; // Очищаем контейнер

        if (films.length === 0) {
            listContainer.innerHTML = "<p style='color:gray; padding:20px;'>Ничего не найдено</p>";
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

    // 4. Категории (как у тебя)
    const categories = {
        "Мультфильмы": ["Cars"],
        "Аниме": ["Invincible"],
        "Хорроры": ["Super_Natural"],
        "Фантастика": ["Invincible", "Super_Natural"],
        "Психологические Триллеры": ["Mr_Robot"],
        "Комедия": ["Home_Alone"],
        "Фэнтези": ["Harry_Potter"],
        "Драма": ["The_Rookie"]
    };

    const categorySelect = document.createElement("select");
    function applyCategoryStyles(isMobile) {
        const styles = isMobile ? {
            position: "relative", top: "200px", width: "340px", height: "50px", left: "6%",
            borderRadius: "6px", padding: "4px", fontSize: "15px", border: "none"
        } : {
            position: "relative", top: "7px", marginLeft: "40px", height: "32px",
            borderRadius: "6px", padding: "4px", fontSize: "15px", border: "none", background: "#f5f5f5"
        };
        Object.assign(categorySelect.style, styles);
    }
    mq.addEventListener("change", (e) => applyCategoryStyles(e.matches));
    applyCategoryStyles(mq.matches);

    const optAll = document.createElement("option");
    optAll.value = "all";
    optAll.textContent = "Все категории";
    categorySelect.appendChild(optAll);

    for (let cat in categories) {
        const opt = document.createElement("option");
        opt.value = cat;
        opt.textContent = cat;
        categorySelect.appendChild(opt);
    }
    document.body.insertBefore(categorySelect, listContainer);

    // 5. Логика фильтрации (работает прямо с массивом данных, перерисовывая разметку)
    function applyFilters() {
        const query = searchInput.value.toLowerCase().trim();
        const selectedCat = categorySelect.value;

        const filteredFilms = filmsList.filter(film => {
            const inCategory = selectedCat === "all" || categories[selectedCat]?.includes(film.id);
            const matchSearch = film.id.replace(/_/g, " ").toLowerCase().includes(query) || 
                                film.name.toLowerCase().includes(query);
            return inCategory && matchSearch;
        });

        renderFilms(filteredFilms);
    }

    searchInput.addEventListener("input", applyFilters);
    categorySelect.addEventListener("change", applyFilters);

    // Первый рендер всех фильмов при загрузке
    renderFilms(filmsList);
});