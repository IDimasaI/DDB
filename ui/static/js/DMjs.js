export function render(id, el, update) {
    document.querySelector(id).appendChild(el);
    update();
}
export function parseApp() {
    const data_page = document.getElementById('app')?.getAttribute('data-page');
    return data_page ? JSON.parse(data_page) : null;
}