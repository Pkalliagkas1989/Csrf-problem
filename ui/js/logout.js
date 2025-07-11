document.getElementById("logout-link").addEventListener("click", async (e) => {
  e.preventDefault();

  await fetch("http://localhost:8080/forum/api/session/logout", {
    method: "POST",
    credentials: "include", // Ensure cookies are sent
  });

  window.location.href = "/login";
});
