package Events

import (
	"sort"
	"sync"
)

type PluginName string

type EventHandler struct {
	PluginName PluginName
	Priority   int
	Func       func(*Event)
}

type eventHandlers struct {
	handlers []EventHandler
	sync.RWMutex
}

type EventBus struct {
	SyncEvents  map[string]*eventHandlers
	AsyncEvents map[string]*eventHandlers
	sync.RWMutex
}

type Options struct {
}

type Event struct {
	Options Options
	Token   string
	Custom  any
}

// Создает экземпляр событий
func NewEventBus() *EventBus {
	return &EventBus{
		SyncEvents:  make(map[string]*eventHandlers),
		AsyncEvents: make(map[string]*eventHandlers),
	}
}

// Добавляет событие(запись синхронных и ассинхронных событий вручную)
func (eb *EventBus) AddHandler(isAsync bool, eventName string, handler EventHandler) {
	eb.Lock()
	defer eb.Unlock()

	events := eb.SyncEvents
	if isAsync {
		events = eb.AsyncEvents
	}

	if events[eventName] == nil {
		events[eventName] = &eventHandlers{}
	}

	eh := events[eventName]
	eh.Lock()
	defer eh.Unlock()

	eh.handlers = append(eh.handlers, handler)
	sort.SliceStable(eh.handlers, func(i, j int) bool {
		return eh.handlers[i].Priority > eh.handlers[j].Priority
	})
}

// Добавляет синхронное событие
func (eb *EventBus) AddSyncHandler(eventName string, handler EventHandler) {
	eb.AddHandler(false, eventName, handler)
}

// Добавляет асинхронное событие
func (eb *EventBus) AddAsyncHandler(eventName string, handler EventHandler) {
	eb.AddHandler(true, eventName, handler)
}

func (eb *EventBus) getHandlers(isAsync bool, name string) *eventHandlers {
	eb.RLock()
	defer eb.RUnlock()

	if isAsync {
		return eb.AsyncEvents[name]
	}
	return eb.SyncEvents[name]
}

// Вызывает событие синхронно, тоесть выполнение происходит в одном потоке с вызывающим кодом
func (eb *EventBus) EmitSync(event *Event, EventName string) {
	handlers := eb.getHandlers(false, EventName)
	if handlers == nil {
		return
	}

	handlers.RLock()
	defer handlers.RUnlock()

	for _, h := range handlers.handlers {
		h.Func(event)
	}
}

// Вызывает ассинхронное событие, поумолчанию все происходит в одной отдельной горутине.
// Тоесть вызывается отдельная горутина, и все функции обрабатываются последовательно.
func (eb *EventBus) EmitAsync(event *Event, EventName string) {
	handlers := eb.getHandlers(true, EventName)
	if handlers == nil {
		return
	}

	handlers.RLock()
	defer handlers.RUnlock()

	go func() {
		for _, h := range handlers.handlers {
			h.Func(event)
		}
	}()
}

// Вызывает события полностью ассинхронно(каждая функция через собственную горутину)
func (eb *EventBus) EmitParallel(e *Event, EventName string) {
	handlers := eb.getHandlers(true, EventName)
	if handlers == nil {
		return
	}

	handlers.RLock()
	defer handlers.RUnlock()

	for _, h := range handlers.handlers {
		go h.Func(e)
	}
}
