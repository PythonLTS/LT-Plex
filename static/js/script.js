document.addEventListener("DOMContentLoaded", async function() {
    try {
        const res = await fetch("/profile-data", { credentials: "include" });
        if (!res.ok) throw new Error("Unauthorized");
        window.addEventListener("pageshow", function (event) {
          if (event.persisted) {
            window.location.reload();
          }
        });
        const data = await res.json();

        document.getElementById("username").textContent = data.username;
        document.getElementById("avatar").src = data.avatar;

        const list = document.getElementById("List");
        list.innerHTML = ""; 

        if (data.favorites && data.favorites.length > 0) {
            data.favorites.forEach(rawName => {
                const name = rawName.includes("_") ? rawName.replace(/_/g, " ") : rawName;

                const div = document.createElement("div");
                div.className = "movie-item";
                div.textContent = name;

                const img = document.createElement("img");
                img.src = `films/${rawName}/${rawName}.png`;
                img.alt = name;
                img.className = "movie-banner";

                const buttonsDiv = document.createElement("div");
                buttonsDiv.className = "movie-buttons";

                const watchBtn = document.createElement("button");
                watchBtn.className = "watch";
                watchBtn.textContent = "Смотреть";
                watchBtn.onclick = () => window.location.href = `/films/${rawName}/${rawName}.html`;

                const favoriteBtn = document.createElement("button");
                favoriteBtn.className = "favorite";
                favoriteBtn.textContent = "Удалить из избранных";
                favoriteBtn.onclick = () => {
                    fetch('rmfv', {
                        method: "POST",
                        credentials: "include",
                        headers: { "Content-Type": "application/json" },
                        body: JSON.stringify({ movie: rawName })
                    });
                    div.remove();
                };

                buttonsDiv.appendChild(watchBtn);
                buttonsDiv.appendChild(favoriteBtn);

                div.prepend(img);
                div.appendChild(buttonsDiv);

                list.appendChild(div);
            });

            
        } else {
            list.innerHTML = "<p style='color:white;font-size:21px;font-weight:bold;margin:20px'>Нет Избранных фильмов<br>Перейдите во вкладку 'Список фильмов'<br>И выберите понравившийся</p>";
        }

    } catch (err) {
        window.location.href = "/sign";
        return;
    }

    const logoutBtn = document.getElementById("logout");
    logoutBtn.addEventListener("click", function(e) {
        e.preventDefault();
        if (confirm("Подтвердите")) {
            window.location.href = "/lt";
        }
    });
    const burger = document.getElementById('burger');
    const itemBar = document.getElementById('item-bar');

    burger.addEventListener('click', () => {
        burger.classList.toggle('open');
        itemBar.classList.toggle('open');
    });

    document.querySelectorAll('#item-bar a').forEach(link => {
        link.addEventListener('click', () => {
            if (window.innerWidth <= 768) {
                burger.classList.remove('open');
                itemBar.classList.remove('open');
            }
        });
    });
});
