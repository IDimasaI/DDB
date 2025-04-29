import * as DMjs from './DMjs.js';


async function GetData() {
    const res = await fetch('/get', { method: 'GET' });
    const data = await res.json();
    console.log(data);
}

async function SentData() {


    const input = document.getElementById("input").value

    const send = JSON.stringify({ data: input })

    const res = await fetch('/set', {
        method: 'POST',
        body: send,
        headers: { 'Content-Length': '0' }
    });
    const data = await res.json();
    console.log(data);
}




const section = document.createElement('section');
section.classList.add('bg-slate-900', 'text-white', 'w-full', 'py-8');

const button1 = document.createElement('button');
button1.addEventListener('click', () => { GetData(); });
button1.textContent = 'Отправить Get запрос';

const button2 = document.createElement('button');
button2.addEventListener('click', () => { SentData(); });
button2.textContent = 'отправить Post запрос';

const br = document.createElement("br")

const input = document.createElement("input")
input.id = "input"
input.setAttribute("value", "test")


section.appendChild(button1);
section.appendChild(button2);
section.appendChild(br)
section.appendChild(input)

function update() {

}




DMjs.render('#root', section, update);


//
//
//
//  <template>
//    <section>
//        <button onClick={() => {GetData();}}>Отправить Get запрос</button>
//        <button onClick={() => {SentData();}}>отправить Post запрос</button>
//        <br>
//        <input id="input" value="test" />
//    </section>
//  </template>;
//
//
//
//