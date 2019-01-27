window.emit = function(o) {
  window.external.invoke(JSON.stringify(o));
}

window.addEventListener("load", function() {
  emit(["pageload"]);
});