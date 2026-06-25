
// Объявление функций, которые ты просил связать
async function scanStorage() {
    const display = document.getElementById("storageDisplay");
    try {
        const response = await fetch('/api/StorageInfo?mode=list');
        if (!response.ok) throw new Error();
        const data = await response.json();
        
        setFreeStorage(data.freeSpace || "--");
        display.innerHTML = "";

        const categories = [
            { title: "Фильмы", count: data.movies, type: "movie" },
            { title: "Сериалы", count: data.series, type: "series" },
            { title: "Каналы", count: data.channels, type: "channel" }
        ];

        categories.forEach(cat => {
            const tile = document.createElement("div");
            tile.className = "storage-tile";
            tile.innerHTML = `
                <div class="tile-content">
                    <span class="tile-title">${cat.title}</span>
                    <span class="tile-count">${cat.count ?? 0}</span>
                </div>
            `;
            tile.addEventListener("click", () => openContentDrawer(cat.title, cat.type));
            display.appendChild(tile);
        });
    } catch (e) {
        display.innerHTML = `<div class="error">Ошибка загрузки данных</div>`;
    }
}

async function openContentDrawer(title, modeType) {
    const overlay = document.getElementById("storageOverlay");
    const drawerTitle = document.getElementById("drawerTitle");
    const drawerContent = document.getElementById("drawerContent");

    drawerTitle.textContent = title;
    drawerContent.innerHTML = "<div>Загрузка...</div>";
    overlay.style.display = "flex"; // Показываем окно по центру

    try {
        const response = await fetch(`/api/StorageInfo?mode=${modeType}`);
        const files = await response.json(); 

        if (!files || files.length === 0) {
            drawerContent.innerHTML = "<div>Папка пуста</div>";
            return;
        }
        drawerContent.innerHTML = "";

        files.forEach(file => {
            const fileRow = document.createElement("div");
            fileRow.className = "file-item";
            fileRow.innerHTML = `
                <div class="file-info-block">
                    <img src="${file.logo}" class="file-preview" alt="cover">
                    <span class="file-name">${file.name.replace(/_/g, ' ')}</span>
                </div>
                <button class="btn-delete" onclick="deleteFile('${modeType}', '${file.name}', this)">Удалить</button>
            `;
            drawerContent.appendChild(fileRow);
        });
    } catch (e) {
        drawerContent.innerHTML = "<div>Ошибка загрузки списка файлов</div>";
    }
}

async function deleteFile(modeType, fileName, btn) {
    if (!confirm(`Удалить ${fileName}?`)) return;
    btn.disabled = true;
    btn.textContent = "...";

    try {
        const response = await fetch(`/api/deletePackage?mode=${modeType}&name=${fileName}`, { method: 'DELETE' });
        if (response.ok) {
            btn.closest('.file-item').remove();
            if (document.getElementById("drawerContent").children.length === 0) {
                document.getElementById("drawerContent").innerHTML = "<div>Папка пуста</div>";
            }
            scanStorage();
        } else {
            alert("Ошибка сервера");
            btn.disabled = false;
            btn.textContent = "Удалить";
        }
    } catch (e) {
        alert("Ошибка сети");
        btn.disabled = false;
        btn.textContent = "Удалить";
    }
}

function closeContentDrawer() {
    document.getElementById("storageOverlay").style.display = "none";
}
    // Сюда пойдет логика сканирования хранилища
function setFreeStorage(capacity) {
    const storageInfoEl = document.querySelector('.storage-info');
    if (storageInfoEl) storageInfoEl.textContent = `Доступно место на устройстве : ${capacity}`;
}

function createLTpack() {
    const overlay = document.getElementById("storageOverlay");
    const drawerTitle = document.getElementById("drawerTitle");
    const drawerContent = document.getElementById("drawerContent");

    // Меняем заголовок модалки
    drawerTitle.textContent = "Создание Пакета";
    drawerContent.innerHTML = "";

    // Данные для сборки пакета (внутреннее состояние)
    const packData = {
        type: 'movie', // по умолчанию
        name: '',
        cover: null,
        items: [] // для серий или видео канала
    };

    // Создаем контейнер для конструктора пакетов
    const container = document.createElement("div");
    container.className = "pack-builder-container";

    // Функция для перерендеринга динамической нижней части (в зависимости от типа)
    function renderDynamicZone() {
        const zone = container.querySelector(".pack-dynamic-zone");
        if (!zone) return;
        zone.innerHTML = "";

        if (packData.type === 'movie') {
            // --- МАКЕТ ФИЛЬМА ---
            zone.className = "pack-dynamic-zone type-movie";
            zone.innerHTML = `
                <div class="upload-dropzone full-zone" id="movieVideoDrop">
                    <span class="dropzone-text">${packData.items[0] ? packData.items[0].name : 'Загрузить Видео'}</span>
                    <span class="dropzone-hint">${packData.items[0] ? '(Кликните, чтобы удалить)' : '(после загрузки здесь появится название файла)'}</span>
                </div>
            `;
            
            const drop = zone.querySelector("#movieVideoDrop");
            if(packData.items[0]) drop.classList.add("has-file");
            
            drop.addEventListener("click", () => {
                if (packData.items[0]) {
                    packData.items = [];
                    renderDynamicZone();
                } else {
                    triggerFileSelect((file) => {
                        packData.items = [{ name: file.name, file: file }];
                        renderDynamicZone();
                    }, "video/*");
                }
            });

        } else {
            // --- МАКЕТ СЕРИАЛА И КАНАЛА ---
            zone.className = "pack-dynamic-zone type-split";
            const isSeries = packData.type === 'series';

            // Левая колонка: список сезонов/видео
            const sidebar = document.createElement("div");
            sidebar.className = "pack-split-sidebar";

            if (isSeries) {
                // Логика сериала: Группировка по Сезонам
                packData.items.forEach((season, sIdx) => {
                    const itemRow = document.createElement("div");
                    itemRow.className = `split-item-nav ${packData.activeSeason === sIdx ? 'active' : ''}`;
                    itemRow.innerHTML = `<span>Сезон ${sIdx + 1}</span><button class="btn-remove-split">&times;</button>`;
                    
                    itemRow.addEventListener("click", (e) => {
                        if(e.target.classList.contains("btn-remove-split")) {
                            e.stopPropagation();
                            packData.items.splice(sIdx, 1);
                            if(packData.activeSeason >= packData.items.length) packData.activeSeason = Math.max(0, packData.items.length - 1);
                            renderDynamicZone();
                        } else {
                            packData.activeSeason = sIdx;
                            renderDynamicZone();
                        }
                    });
                    sidebar.appendChild(itemRow);
                });

                const btnAdd = document.createElement("div");
                btnAdd.className = "split-item-nav btn-add-split";
                btnAdd.innerHTML = `<span>Добавить</span><span class="plus-icon">+</span>`;
                btnAdd.addEventListener("click", () => {
                    packData.items.push({ episodes: [] });
                    packData.activeSeason = packData.items.length - 1;
                    renderDynamicZone();
                });
                sidebar.appendChild(btnAdd);

            } else {
                // Логика Канала: Просто список загруженных видео
                packData.items.forEach((video, vIdx) => {
                    const itemRow = document.createElement("div");
                    itemRow.className = "split-item-nav channel-video-item";
                    itemRow.innerHTML = `<span title="${video.name}">${video.name}</span><button class="btn-remove-split">&times;</button>`;
                    itemRow.querySelector(".btn-remove-split").addEventListener("click", (e) => {
                        e.stopPropagation();
                        packData.items.splice(vIdx, 1);
                        renderDynamicZone();
                    });
                    sidebar.appendChild(itemRow);
                });

                const btnAdd = document.createElement("div");
                btnAdd.className = "split-item-nav btn-add-split";
                btnAdd.innerHTML = `<span>Добавить видео</span>`;
                btnAdd.addEventListener("click", () => {
                    triggerFileSelect((file) => {
                        packData.items.push({ name: file.name, file: file });
                        renderDynamicZone();
                    }, "video/*");
                });
                sidebar.appendChild(btnAdd);
            }

            // Правая колонка: Контент выбранного элемента
            const contentArea = document.createElement("div");
            contentArea.className = "pack-split-main";

            if (isSeries) {
                const currentSeasonIdx = packData.activeSeason ?? 0;
                const currentSeason = packData.items[currentSeasonIdx];

                if (currentSeason) {
                    contentArea.innerHTML = `
                        <div class="season-header">Выбран ${currentSeasonIdx + 1}</div>
                        <div class="upload-dropzone embedded-zone" id="episodeDrop">
                            <span class="dropzone-text">Добавить Серию</span>
                            <span class="dropzone-hint">Или если уже загружена то название файла и сделать кликабельной чтоб удалить</span>
                        </div>
                        <div class="episode-list"></div>
                    `;

                    const epListContainer = contentArea.querySelector(".episode-list");
                    currentSeason.episodes.forEach((ep, epIdx) => {
                        const epEl = document.createElement("div");
                        epEl.className = "uploaded-ep-link";
                        epEl.textContent = ep.name;
                        epEl.addEventListener("click", () => {
                            currentSeason.episodes.splice(epIdx, 1);
                            renderDynamicZone();
                        });
                        epListContainer.appendChild(epEl);
                    });

                    contentArea.querySelector("#episodeDrop").addEventListener("click", () => {
                        triggerFileSelect((file) => {
                            currentSeason.episodes.push({ name: file.name, file: file });
                            renderDynamicZone();
                        }, "video/*");
                    });
                } else {
                    contentArea.innerHTML = `<div class="split-empty-state">Добавьте сезон слева</div>`;
                }
            } else {
                // Для канала справа выводим форму названия видео
                contentArea.innerHTML = `
                    <div class="channel-inputs">
                        <div class="input-block">
                            <input type="text" id="chanVideoName" placeholder="Название видео" class="pack-input">
                        </div>
                        <div class="upload-dropzone embedded-zone" id="chanVideoFileDrop">
                            <span class="dropzone-text">Загрузить видео</span>
                        </div>
                    </div>
                `;
                
                const inpName = contentArea.querySelector("#chanVideoName");
                inpName.addEventListener("input", (e) => {
                    // Можно сохранять кастомное имя, если требуется
                });

                contentArea.querySelector("#chanVideoFileDrop").addEventListener("click", () => {
                    triggerFileSelect((file) => {
                        const customName = inpName.value.trim() || file.name;
                        packData.items.push({ name: customName, file: file });
                        inpName.value = "";
                        renderDynamicZone();
                    }, "video/*");
                });
            }

            zone.appendChild(sidebar);
            zone.appendChild(contentArea);
        }
    }

    // Хелпер создания скрытого инпута выбора файлов
    function triggerFileSelect(callback, accept = "*/*") {
        const fileInput = document.createElement("input");
        fileInput.type = "file";
        fileInput.accept = accept;
        fileInput.addEventListener("change", (e) => {
            if (e.target.files.length > 0) callback(e.target.files[0]);
        });
        fileInput.click();
    }

    // Рендерим базовую сетку шапки (Обложка, Тип, Название)
    container.innerHTML = `
        <div class="pack-grid-top">
            <div class="pack-cover-box" id="packCover">
                <span class="cover-text">Обложка/пусто</span>
            </div>
            <div class="pack-meta-box">
                <input type="text" id="packTitle" placeholder="Название" class="pack-input">
                <div class="pack-type-selector">
                    <span class="type-label">Тип :</span>
                    <select id="packTypeSelect" class="pack-select">
                        <option value="movie">Фильм</option>
                        <option value="series">Сериал</option>
                        <option value="channel">Канал</option>
                    </select>
                </div>
            </div>
        </div>
        <div class="pack-dynamic-zone"></div>
        <div class="pack-footer-actions">
            <button class="pack-btn btn-cancel" id="packBtnCancel">Отменить</button>
            <button class="pack-btn btn-submit" id="packBtnCreate">Создать</button>
        </div>
    `;

    drawerContent.appendChild(container);

    // Вешаем события на верхнюю статичную часть
    const packTypeSelect = container.querySelector("#packTypeSelect");
    packTypeSelect.addEventListener("change", (e) => {
        packData.type = e.target.value;
        packData.items = []; // Сброс файлов при смене типа
        if(packData.type === 'series') packData.activeSeason = 0;
        renderDynamicZone();
    });

    container.querySelector("#packTitle").addEventListener("input", (e) => {
        packData.name = e.target.value;
    });

    container.querySelector("#packCover").addEventListener("click", () => {
        triggerFileSelect((file) => {
            packData.cover = file;
            const coverBox = container.querySelector("#packCover");
            coverBox.style.backgroundImage = `url(${URL.createObjectURL(file)})`;
            coverBox.style.backgroundSize = "cover";
            coverBox.style.backgroundPosition = "center";
            coverBox.querySelector(".cover-text").style.display = "none";
        }, "image/*");
    });

    container.querySelector("#packBtnCancel").addEventListener("click", closeContentDrawer);
    
    container.querySelector("#packBtnCreate").addEventListener("click", async () => {
        if (!packData.name.trim()) return alert("Введите название пакета!");
        
        // Показываем индикатор загрузки на кнопке
        const btnSubmit = container.querySelector("#packBtnCreate");
        const originalText = btnSubmit.textContent;
        btnSubmit.disabled = true;
        btnSubmit.textContent = "Упаковка...";

        try {
            // Инициализируем JSZip
            const zip = new JSZip();
            let meta = {
                Name: packData.name,
                Type: packData.type
            };

            switch (packData.type) {
                case "movie":
                    meta.Quantity = 1;
                    break;

                case "channel":
                    meta.Quantity = packData.items.length;
                    break;

                case "series":
                    meta.Seasons = packData.items.length;
                    meta.EpisodesPerSeason = packData.items.map(
                        season => season.episodes.length
                    );
                    break;
            }

            zip.file("meta.json", JSON.stringify(meta, null, 2));
            // 1. Добавляем обложку (если она загружена)
            if (packData.cover) {
                // Извлекаем расширение файла обложки (обычно jpg или png)
                const coverExt = packData.cover.name.split('.').pop();
                zip.file(`${packData.name}.${coverExt}`, packData.cover);
            }

            // 2. Формируем структуру и добавляем видео-файлы
            if (packData.type === 'movie') {
                // Для фильмов просто кладем видео в корень архива
                if (packData.items.length === 0) {
                    throw new Error("Добавьте видео-файл для фильма!");
                }
                const movieFile = packData.items[0].file;
                zip.file(movieFile.name, movieFile);

            } else if (packData.type === 'series') {
                // Для сериалов создаем папки: Сезон 1, Сезон 2 и т.д.
                if (packData.items.length === 0) {
                    throw new Error("Добавьте хотя бы один сезон и серии к нему!");
                }
                
                packData.items.forEach((season, sIdx) => {
                    const seasonFolderName = `Season${sIdx + 1}`;
                    // Создаем виртуальную папку в ZIP
                    const seasonFolder = zip.folder(seasonFolderName);
                    
                    season.episodes.forEach(ep => {
                        seasonFolder.file(ep.file.name, ep.file);
                    });
                });

            } else if (packData.type === 'channel') {
                // Для каналов кладем все видео в корень, но можем переименовать файлы,
                // если пользователь ввел кастомное имя в интерфейсе
                if (packData.items.length === 0) {
                    throw new Error("Добавьте хотя бы одно видео для канала!");
                }
                if (packData.items.length >= 30){
                    alert("Осторожно Возможны зависания\nизза обьема пакета");
                }
                packData.items.forEach(item => {
                    const originalExt = item.file.name.split('.').pop();
                    // Если имя изменили вручную, сохраняем с новым именем + старое расширение
                    const finalName = item.name.endsWith(`.${originalExt}`) 
                        ? item.name 
                        : `${item.name}.${originalExt}`;
                        
                    zip.file(finalName, item.file);
                });
            }

            // 3. Генерируем ZIP-архив в виде бинарного блоба (Blob)
            // и отслеживаем прогресс в консоли (полезно для тяжелых видео)
            const contentBlob = await zip.generateAsync({ type: "blob" }, (metadata) => {
                btnSubmit.textContent = `Упаковка: ${Math.floor(metadata.percent)}%`;
            });

            // 4. Скачивание готового файла с расширением .ltpack
            // Заменяем пробелы на подчеркивания в названии, чтобы имя было валидным
            const safeName = packData.name.trim().replace(/\s+/g, '_');
            const fileName = `${safeName}.ltpack`;

            const link = document.createElement("a");
            link.href = URL.createObjectURL(contentBlob);
            link.download = fileName;
            
            // Триггерим скачивание в браузере
            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);
            
            // Освобождаем память
            URL.revokeObjectURL(link.href);

            alert(`Пакет "${fileName}" успешно создан и скачивается!`);
            closeContentDrawer();

        } catch (error) {
            console.error("Ошибка при сборке пакета:", error);
            alert(error.message || "Произошла ошибка при генерации .ltpack");
        } finally {
            // Возвращаем кнопку в исходное состояние
            btnSubmit.disabled = false;
            btnSubmit.textContent = originalText;
        }
    });

    // Инициализируем нижнюю зону
    renderDynamicZone();

    // Открываем модалку
    overlay.style.display = "flex";
}

function uploadLTpack() {
    console.log("Вызвана функция: uploadLTpack()");

    const input = document.createElement("input");
    input.type = "file";
    input.accept = ".ltpack";

    input.onchange = async () => {
        const file = input.files?.[0];

        if (!file) {
            return;
        }

        if (!file.name.toLowerCase().endsWith(".ltpack")) {
            alert("Можно выбрать только ZIP-файл");
            return;
        }

        try {
            const response = await fetch("/api/addPackage", {
                method: "POST",
                headers: {
                    "Content-Type": "application/zip",
                    "X-File-Name": encodeURIComponent(file.name) // если нужно передать имя файла
                },
                body: file
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }

            const result = await response.text();

            console.log("Пакет успешно загружен:", result);
        } catch (err) {
            console.error("Ошибка загрузки:", err);
            alert("Ошибка загрузки пакета");
        }
    };

    input.click();
}

// Ждем полной загрузки DOM дерева
document.addEventListener("DOMContentLoaded", () => {
    scanStorage();
    document.getElementById("back").addEventListener("click", () => {
        window.location.href = "/profile";
    });
    document.getElementById("btnCreate").addEventListener("click",createLTpack);
    document.getElementById("btnUpload").addEventListener("click",uploadLTpack);
    document.getElementById("btnScan").addEventListener("click", scanStorage);
    document.getElementById("closeDrawer").addEventListener("click", closeContentDrawer);
    document.getElementById("storageOverlay").addEventListener("click", (e) => {
        if (e.target.id === "storageOverlay") closeContentDrawer();
    });
});