import * as DMjs from './DMjs.js';

let count = 0;

//секция + h
const section = document.createElement('section');
section.classList.add('bg-slate-900', 'text-white', 'w-full', 'py-8');

const h1 = document.createElement('h1');
h1.textContent = 'Значение счетчика: ';

const span = document.createElement('span');
span.textContent = count;

h1.appendChild(span);
section.appendChild(h1);

//Кнопка

const button = document.createElement('button');
button.addEventListener('click', () => { console.log(DMjs.parseApp()); count++; update(); });

section.appendChild(button);

/**
 * #### Обновление всех элементов содержащих шаблонные строки.
 * TDO: при нахождении компилятором шаблонной строки выносить ее в отдельную функцию, и запихивать сюда
 */
function update() {
    button.textContent = `Кликов: ${count + 1}`;
    span.textContent = count;

}







DMjs.render('#root', section, update);

//
//
//
//
//  <script>
//    let count = 0;
//  </script>
//
//
//  <template>
//    <section className="bg-slate-900 text-white w-full py-8">
//        <h1>Значение счетчика: {count+1}</h1>
//        <button onClick={() => { count++; update(); }}>Кликов: <span>{count}</span></button>
//    </section>
//  </template>;
//
