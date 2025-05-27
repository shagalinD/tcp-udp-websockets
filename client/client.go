package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Sensor struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Location string  `json:"location"`
	Value    float64 `json:"value"`
	Unit     string  `json:"unit"`
}

type EmergencyAlert struct {
	SensorID  string    `json:"sensor_id"`
	Message   string    `json:"message"`
	Level     string    `json:"level"`
	Timestamp time.Time `json:"timestamp"`
}

type ServerConfig struct {
	ID      string `json:"id"`
	Address string `json:"address"`
	TCPPort string `json:"tcp_port"`
	UDPPort string `json:"udp_port"`
	WSPort  string `json:"ws_port"`
}

type EventCache struct {
	mu     sync.Mutex
	Events []string
	Max    int
}

func (c *EventCache) Add(event string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.Events) >= c.Max {
		c.Events = c.Events[1:]
	}
	c.Events = append(c.Events, event)
}

func (c *EventCache) Get() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Events
}

var (
	servers       []ServerConfig
	currentServer *ServerConfig
	eventCache    = EventCache{Max: 100}
	sensors       = make(map[string]Sensor)
	reader        = bufio.NewReader(os.Stdin)
)

func main() {
	fmt.Println("EcoMonitor - Система контроля загрязнения воздуха")
	fmt.Println("===============================================")

	for {
		fmt.Println("\nМеню:")
		fmt.Println("1. Добавить сервер")
		fmt.Println("2. Удалить сервер")
		fmt.Println("3. Подключиться к серверу")
		fmt.Println("4. Просмотреть список серверов")
		fmt.Println("5. Просмотреть историю событий")
		fmt.Println("6. Выход")

		fmt.Print("Выберите действие: ")
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			addServer()
		case "2":
			removeServer()
		case "3":
			connectToServer()
		case "4":
			listServers()
		case "5":
			showEventHistory()
		case "6":
			fmt.Println("Выход из программы...")
			return
		default:
			fmt.Println("Неверный выбор, попробуйте снова.")
		}
	}
}

func addServer() {
	fmt.Print("Введите адрес сервера (например, localhost): ")
	address, _ := reader.ReadString('\n')
	address = strings.TrimSpace(address)

	fmt.Print("Введите TCP порт: ")
	tcpPort, _ := reader.ReadString('\n')
	tcpPort = strings.TrimSpace(tcpPort)

	fmt.Print("Введите UDP порт: ")
	udpPort, _ := reader.ReadString('\n')
	udpPort = strings.TrimSpace(udpPort)

	fmt.Print("Введите WebSocket порт: ")
	wsPort, _ := reader.ReadString('\n')
	wsPort = strings.TrimSpace(wsPort)

	server := ServerConfig{
		ID:      fmt.Sprintf("server-%d", len(servers)+1),
		Address: address,
		TCPPort: tcpPort,
		UDPPort: udpPort,
		WSPort:  wsPort,
	}

	servers = append(servers, server)
	fmt.Printf("Сервер %s добавлен успешно.\n", server.ID)
}

func removeServer() {
	if len(servers) == 0 {
		fmt.Println("Нет доступных серверов для удаления.")
		return
	}

	listServers()
	fmt.Print("Введите ID сервера для удаления: ")
	id, _ := reader.ReadString('\n')
	id = strings.TrimSpace(id)

	for i, server := range servers {
		if server.ID == id {
			servers = append(servers[:i], servers[i+1:]...)
			fmt.Println("Сервер удален успешно.")
			return
		}
	}

	fmt.Println("Сервер с указанным ID не найден.")
}

func listServers() {
	if len(servers) == 0 {
		fmt.Println("Нет доступных серверов.")
		return
	}

	fmt.Println("\nСписок серверов:")
	for _, server := range servers {
		fmt.Printf("ID: %s, Адрес: %s, TCP: %s, UDP: %s, WS: %s\n",
			server.ID, server.Address, server.TCPPort, server.UDPPort, server.WSPort)
	}
}

func connectToServer() {
	if len(servers) == 0 {
		fmt.Println("Нет доступных серверов. Пожалуйста, добавьте сервер сначала.")
		return
	}

	listServers()
	fmt.Print("Введите ID сервера для подключения: ")
	id, _ := reader.ReadString('\n')
	id = strings.TrimSpace(id)

	for _, server := range servers {
		if server.ID == id {
			currentServer = &server
			break
		}
	}

	if currentServer == nil {
		fmt.Println("Сервер с указанным ID не найден.")
		return
	}

	fmt.Printf("Подключаемся к серверу %s...\n", currentServer.ID)

	// Подключаемся к TCP для получения списка датчиков
	tcpAddr := fmt.Sprintf("%s:%s", currentServer.Address, currentServer.TCPPort)
	conn, err := net.Dial("tcp", tcpAddr)
	if err != nil {
		log.Printf("Ошибка подключения TCP: %v\n", err)
		return
	}
	defer conn.Close()

	// Запрашиваем список датчиков
	fmt.Fprintf(conn, "GET_SENSORS\n")

	// Читаем ответ
	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		var availableSensors []Sensor
		err := json.Unmarshal([]byte(scanner.Text()), &availableSensors)
		if err != nil {
			log.Printf("Ошибка разбора JSON: %v\n", err)
			return
		}

		fmt.Println("\nДоступные датчики:")
		for _, sensor := range availableSensors {
			fmt.Printf("ID: %s, Имя: %s, Местоположение: %s\n", sensor.ID, sensor.Name, sensor.Location)
		}

		fmt.Print("\nВведите ID датчиков для подключения (через запятую): ")
		sensorIDs, _ := reader.ReadString('\n')
		sensorIDs = strings.TrimSpace(sensorIDs)
		selectedIDs := strings.Split(sensorIDs, ",")

		// Запускаем UDP для получения данных датчиков
		go startUDPListener(currentServer.Address, currentServer.UDPPort, selectedIDs)

		// Запускаем WebSocket для экстренных оповещений
		go startWebSocketListener(currentServer.Address, currentServer.WSPort)

		// Основной цикл для отображения данных
		displayDataLoop(selectedIDs)
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Ошибка чтения TCP: %v\n", err)
	}
}

func startUDPListener(address, port string, sensorIDs []string) {
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%s", address, port))
	if err != nil {
		log.Printf("Ошибка разрешения UDP адреса: %v\n", err)
		return
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Printf("Ошибка запуска UDP слушателя: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Printf("UDP слушатель запущен на %s:%s\n", address, port)

	buffer := make([]byte, 1024)
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Ошибка чтения UDP: %v\n", err)
			continue
		}

		var sensorData Sensor
		err = json.Unmarshal(buffer[:n], &sensorData)
		if err != nil {
			log.Printf("Ошибка разбора данных датчика: %v\n", err)
			continue
		}

		// Проверяем, что этот датчик выбран пользователем
		for _, id := range sensorIDs {
			if id == strings.TrimSpace(sensorData.ID) {
				sensors[sensorData.ID] = sensorData
				eventCache.Add(fmt.Sprintf("[UDP] Датчик %s: %.2f %s", sensorData.Name, sensorData.Value, sensorData.Unit))
				break
			}
		}
	}
}

func startWebSocketListener(address, port string) {
	u := url.URL{Scheme: "ws", Host: fmt.Sprintf("%s:%s", address, port), Path: "/alerts"}
	log.Printf("Подключаемся к WebSocket %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Printf("Ошибка подключения WebSocket: %v\n", err)
		return
	}
	defer c.Close()

	fmt.Printf("WebSocket подключен к %s:%s\n", address, port)

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Printf("Ошибка чтения WebSocket: %v\n", err)
			return
		}

		var alert EmergencyAlert
		err = json.Unmarshal(message, &alert)
		if err != nil {
			log.Printf("Ошибка разбора оповещения: %v\n", err)
			continue
		}

		eventCache.Add(fmt.Sprintf("[ALERT] %s: %s (Уровень: %s)", alert.SensorID, alert.Message, alert.Level))
		fmt.Printf("\n=== ЭКСТРЕННОЕ ОПОВЕЩЕНИЕ ===\nДатчик: %s\nУровень: %s\nСообщение: %s\nВремя: %s\n=======================\n",
			alert.SensorID, alert.Level, alert.Message, alert.Timestamp.Format("2006-01-02 15:04:05"))
	}
}

func displayDataLoop(sensorIDs []string) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fmt.Println("\nТекущие показания датчиков:")
			for _, id := range sensorIDs {
				if sensor, ok := sensors[strings.TrimSpace(id)]; ok {
					fmt.Printf("%s (%s): %.2f %s\n", sensor.Name, sensor.Location, sensor.Value, sensor.Unit)
				} else {
					fmt.Printf("%s: данные отсутствуют\n", id)
				}
			}
		}
	}
}

func showEventHistory() {
	events := eventCache.Get()
	if len(events) == 0 {
		fmt.Println("История событий пуста.")
		return
	}

	fmt.Println("\nПоследние события:")
	for _, event := range events {
		fmt.Println(event)
	}
}