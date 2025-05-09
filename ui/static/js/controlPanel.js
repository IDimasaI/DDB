import * as DMjs from './DMjs.js';


async function GetData() {
    const res = await fetch('/get', { method: 'GET' });
    const data = await res.json();
    console.log(data);
}

async function SentData() {
    for (let i = 0; i < 1; i++) {
        const input = document.getElementById("input").value
        const input2 = document.getElementById("input2").value

        const send = JSON.stringify({
            NameBD: "docs",
            NameTable: "go",
            Data: {
                [input]: {
                    test: input2,
                    input: "test",
                    inDB: ["car", "dog"]
                }
            }
        })

        const res = await fetch('/set', {
            method: 'POST',
            body: send,
            headers: { 'Content-Length': '0' }
        });
        let data = await res.json();
        data = {
            ...data.data,
            row: data.data.row
        }
        console.log(data)
    }
}




const section = document.createElement('section');
section.classList.add('bg-slate-900', 'text-white', 'w-full', 'py-8');

const button1 = document.createElement('button');
button1.textContent = 'Отправить Get запрос';
button1.addEventListener('click', GetData);

const button2 = document.createElement('button');
button2.textContent = 'Отправить Post запрос';
button2.addEventListener('click', SentData);

const input = document.createElement('input');
input.id = 'input';
input.value = 'test';
input.classList.add('border', 'border-gray-300');

const input2 = document.createElement('input')
input2.id = 'input2'
input2.value = '123'
input2.classList.add('border', 'border-gray-300')

section.append(
    button1,
    document.createElement('br'),
    button2,
    document.createElement('br'),
    input,
    document.createElement('br'),
    input2
);

function update() {

}

input.addEventListener('input', update);
input2.addEventListener('input', update)

DMjs.render('#root', section, update);

//                ТЕСТОВОЕ ПРЕДСТАВЛЕНИЕ ШАБЛОНА
//
//  <template>
//    <section>
//        <button onClick={() => {GetData();}}>Отправить Get запрос</button>
//        <br>
//        <button onClick={() => {SentData();}}>отправить Post запрос</button>
//        <br>
//        <input id="input" value="test" class="border border-gray-300" />
//        <p>{{value}}</p>
//    </section>
//  </template>;
