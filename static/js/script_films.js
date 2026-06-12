document.addEventListener("DOMContentLoaded", async function () {

    document.getElementById("back").addEventListener("click", () => {
        window.location.href = "/profile";
    });

    let data = null;
    try {
        const res = await fetch("/profile-data", { credentials: "include" });
        if (!res.ok) throw new Error("Unauthorized");
        data = await res.json();
    } catch (e) {
        return;
    }

    const userFavorites = data.favorites || [];
    const mq = window.matchMedia("(max-width: 600px)");
    const filmBlocks = document.querySelectorAll("#list > div");


    filmBlocks.forEach(block => {
        const filmID = block.id;
        const favBtn = block.querySelector(".favorite");
        const watchBtn = block.querySelector(".watch");

        watchBtn.addEventListener("click", () => {
            window.location.href = `/films/${filmID}/${filmID}.html`;
        });

        const updateFavButton = () => {
            if (userFavorites.includes(filmID)) {
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
                    body: JSON.stringify({ movie: filmID })
                });

                if (res.ok) {
                    if (isAdded) {
                        const i = userFavorites.indexOf(filmID);
                        if (i !== -1) userFavorites.splice(i, 1);
                    } else {
                        userFavorites.push(filmID);
                    }
                    updateFavButton();
                }
            } catch (err) {
                console.error("Ошибка работы с избранным:", err);
            }
        });
    });



    const searchInput = document.createElement("input");
    searchInput.placeholder = "Поиск фильмов...";
    searchInput.id = "search";


    function applySearchStyles(isMobile) {
        if (isMobile) {
            Object.assign(searchInput.style, {
                position:"relative",
                left:"6%",
                top:"160px",
                width: "340px",
                height: "50px",
                fontSize: "20px",
                borderRadius: "6px",
                border: "none",
                paddingLeft: "8px",
                outline: "none",
                background: "#f5f5f5"
            });
        } else {
            Object.assign(searchInput.style, {
                position:"relative",
                left:"35px",
                top:"7px",
                margin: "10px",
                width: "260px",
                height: "32px",
                fontSize: "18px",
                borderRadius: "6px",
                border: "none",
                paddingLeft: "8px",
                outline: "none",
                background: "#f5f5f5"
            });
        }
    }

    mq.addEventListener("change", (e) => {
        applySearchStyles(e.matches);
    });

    applySearchStyles(mq.matches);

    document.body.insertBefore(searchInput, document.getElementById("list"));


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
        if (isMobile) {
            Object.assign(categorySelect.style, {
                position:"relative",
                top:"200px",
                width:"340px",
                height: "50px",
                left:"6%",
                borderRadius: "6px",
                padding: "4px",
                fontSize: "15px",
                border: "none"
            });
        } else {
            Object.assign(categorySelect.style, {
                position:"relative",
                top:"7px",
                marginLeft: "40px",
                height: "32px",
                borderRadius: "6px",
                padding: "4px",
                fontSize: "15px",
                border: "none",
                background: "#f5f5f5"
            });
        }
    }

    mq.addEventListener("change", (e) => {
        applyCategoryStyles(e.matches);
    });

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

    document.body.insertBefore(categorySelect, document.getElementById("list"));


    function applyFilters() {
        const search = searchInput.value.toLowerCase().trim();
        const selectedCat = categorySelect.value;

        filmBlocks.forEach(block => {
            const id = block.id;

            const inCategory =
                selectedCat === "all" ||
                categories[selectedCat]?.includes(id);

            const matchSearch = id.replace(/_/g, " ").toLowerCase().includes(search);

            block.style.display = (inCategory && matchSearch) ? "" : "none";
        });
    }

    searchInput.addEventListener("input", applyFilters);
    categorySelect.addEventListener("change", applyFilters);

    applyFilters();
});
