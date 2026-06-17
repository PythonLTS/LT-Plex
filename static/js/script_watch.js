document.addEventListener("DOMContentLoaded", () => {
	const seasons = document.querySelectorAll(".season");
	document.getElementById("back").addEventListener("click", () => {
        window.history.back();
    });
    const nameElem = document.getElementsByClassName('movie-title')[0];
    const name = encodeURIComponent(nameElem.textContent.trim());
    let nameTitle = nameElem.textContent;
	if (nameTitle.includes("_")) {
	    nameTitle = nameTitle.replace(/_/g, " ");
	}

	nameElem.textContent = nameTitle;
    
	seasons.forEach(season => {
		const header = season.querySelector(".season-header");
		const list = season.querySelector(".episode-list");

		header.addEventListener("click", () => {

			document.querySelectorAll(".episode-list").forEach(l => {
				if (l !== list) l.style.display = "none";
			});
			
			
			document.querySelectorAll(".season-header").forEach(h => h.classList.remove("active"));

			
			header.classList.add("active");
			list.style.display = (list.style.display === "block") ? "none" : "block";
		});

	
		season.querySelectorAll("li").forEach(ep => {
			ep.addEventListener("click", () => {

			
				document.querySelectorAll(".episode-list li").forEach(e => e.classList.remove("active"));

				ep.classList.add("active");

				
				document.querySelectorAll(".season-header").forEach(h => h.classList.remove("active"));
				header.classList.add("active");

				const seasonNum = season.dataset.season;
				const epNum = ep.dataset.ep;

				
				const video = document.querySelector("video");
				video.src = `/films/${name}/Season${seasonNum}/Episode${epNum}/Episode${epNum}.mp4`;
				video.load();
				video.play();
			});
		});
	});
});
