document.addEventListener("DOMContentLoaded", async function() {
    try {
        // Запрос данных профиля
        const res = await fetch("/profile-data", { credentials: "include" });
        if (!res.ok) throw new Error("Unauthorized");
        
        // Обработка кнопкой "Назад" в браузере (BFCache)
        window.addEventListener("pageshow", function (event) {
            if (event.persisted) {
                window.location.reload();
            }
        });

        const data = await res.json();
        let filtredName;
        if (data.username.length > 7) {
            filtredName = data.username.slice(0, 7) + "..";
            document.getElementById("username").textContent = filtredName;
        }
        else {
        // Заполнение информации
            document.getElementById("username").textContent = data.username;
        }
        const list = document.getElementById("List");
        list.innerHTML = ""; 

        if (data.favorites && data.favorites.length > 0) {
            data.favorites.forEach(rawName => {
                const name = rawName.replace(/_/g, " ");

                
                const div = document.createElement("div");
                div.className = "movie-item";

                
                const h3 = document.createElement("h3");
                h3.textContent = name;

                
                const img = document.createElement("img");
                img.className = "movie-banner";
                img.src = `/films/${rawName}/${rawName}.jpeg`;
                img.alt = name;
                
                
                img.onerror = () => {
                    img.src = "/s/images/unknown.png";
                };

                
                const buttonsDiv = document.createElement("div");
                buttonsDiv.className = "movie-buttons";

                
                const watchBtn = document.createElement("button");
                watchBtn.className = "watch";
                watchBtn.textContent = "Смотреть";
                watchBtn.onclick = () => window.location.href = `/films/${rawName}/${rawName}.html`;

                
                const favoriteBtn = document.createElement("button");
                favoriteBtn.className = "favorite";
                favoriteBtn.textContent = "Удалить";
                favoriteBtn.onclick = async () => {
                    try {
                        const response = await fetch('/rmfv', {
                            method: "POST",
                            credentials: "include",
                            headers: { "Content-Type": "application/json" },
                            body: JSON.stringify({ movie: rawName })
                        });
                        if (response.ok) {
                            div.remove();
                            // Если удалили последний элемент, выведем заглушку
                            if (list.children.length === 0) {
                                showEmptyMessage(list);
                            }
                        }
                    } catch (e) {
                        console.error("Ошибка удаления:", e);
                    }
                };

                // Сборка структуры
                buttonsDiv.appendChild(watchBtn);
                buttonsDiv.appendChild(favoriteBtn);
                
                div.appendChild(img);
                div.appendChild(h3);
                div.appendChild(buttonsDiv);

                list.appendChild(div);
            });
        } else {
            showEmptyMessage(list);
        }

    } catch (err) {
        console.error(err);
        window.location.href = "/sign";
        return;
    }

    // Вспомогательная функция для генерации красивой заглушки пустого списка
    function showEmptyMessage(container) {
        container.innerHTML = `
            <div class="empty-msg">
                <p style="font-size: 25px;"><strong>Нет избранных фильмов</strong></p>
                <p style="font-size: 17px; margin-top: 10px; color: #aaa;">
                    Перейдите во вкладку "Хранилище", выберите понравившийся контент и нажмите "В избранное".
                </p>
            </div>`;
    }

    // Логика кнопки логаута
    const logoutBtn = document.getElementById("logout");
    if (logoutBtn) {
        logoutBtn.addEventListener("click", function(e) {
            e.preventDefault();
            if (confirm("Вам придется заново вводить данные,Вы уверены?")) {
                window.location.href = "/api/logout";
            }
        });
    }

    // Бургер меню (только для телефонов)
    const burger = document.getElementById('burger');
    const itemBar = document.getElementById('item-bar');

    if (burger && itemBar) {
        burger.addEventListener('click', () => {
            burger.classList.toggle('open');
            itemBar.classList.toggle('open');
        });

        // Закрытие меню при клике на ссылки ТОЛЬКО на мобилках
        document.querySelectorAll('#item-bar a').forEach(link => {
            link.addEventListener('click', () => {
                if (window.innerWidth <= 768) {
                    burger.classList.remove('open');
                    itemBar.classList.remove('open');
                }
            });
        });
    }
});