# EcoMonitor - Система мониторинга качества воздуха

Многофункциональная система для мониторинга показателей окружающей среды с использованием трех сетевых протоколов.

## Особенности

- 🌐 Поддержка трех протоколов:
  - **TCP** для управления подключениями
  - **UDP** для потоковой передачи данных
  - **WebSocket** для экстренных оповещений
- 📊 Генерация данных в реальном времени
- 🔔 Система экстренных уведомлений
- 📂 Локальное кэширование событий
- 🖥 Управление несколькими серверами

## Установка

1. Клонировать репозиторий:
   ```bash
   git clone https://github.com/shagalinD/tcp-udp-websockets
   cd ecomonitor
   ```

2. Установить зависимости:
   ```bash
   go mod tidy
   ```

## Использование

### Запуск сервера
```bash
go run server/main.go
```

Сервер запустит три службы:
- TCP-сервер на порту `:8080`
- UDP-сервер на порту `:8081`
- WebSocket-сервер на порту `:8082`

### Запуск клиента
```bash
go run client/main.go
```

**Пример работы:**
1. Добавьте сервер:
   ```
   Адрес: localhost
   TCP порт: 8080
   UDP порт: 8081
   WS порт: 8082
   ```

2. Подключитесь к серверу
3. Выберите датчики из списка (например: `s1,s2`)
4. Наблюдайте данные в реальном времени

## Структура проекта

```
ecomonitor/
├── client/           # Клиентская часть
│   ├── main.go       # Основной клиентский модуль
├── server/           # Серверная часть
│   ├── main.go       # Основной серверный модуль
└── README.md         # Документация
```

## Конфигурация сервера

Порты по умолчанию:
```text
TCP: 8080
UDP: 8081
WebSocket: 8082
```

Для изменения портов отредактируйте соответствующие константы в коде сервера.

## Генерация данных

- 📈 Данные датчиков обновляются каждые 10 секунд
- 🚨 Экстренные события генерируются случайным образом (~1 раз в 3 минуты)
