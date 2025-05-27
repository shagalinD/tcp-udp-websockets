package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	UDP_PORT = ":8081"
	TCP_PORT = ":8080"
	WS_PORT  = ":8082"
)

type Sensor struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Location string  `json:"location"`
	Unit     string  `json:"unit"`
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
	value    float64
	mu       sync.Mutex
}

type EmergencyAlert struct {
	SensorID  string    `json:"sensor_id"`
	Message   string    `json:"message"`
	Level     string    `json:"level"`
	Timestamp time.Time `json:"timestamp"`
}

var (
	sensors     = make(map[string]*Sensor)
	upgrader    = websocket.Upgrader{}
	wsClients   = make(map[*websocket.Conn]bool)
	wsClientsMu sync.Mutex
)

func main() {
	rand.Seed(time.Now().UnixNano())

	// Инициализация тестовых датчиков
	initializeSensors()

	// Запуск TCP сервера
	go startTCPServer(TCP_PORT)

	// Запуск UDP сервера
	go startUDPServer(UDP_PORT)

	// Запуск HTTP/WebSocket сервера
	http.HandleFunc("/alerts", handleWebSocket)
	log.Fatal(http.ListenAndServe(WS_PORT, nil))
}

func initializeSensors() {
	sensors["s1"] = &Sensor{
		ID:       "s1",
		Name:     "PM2.5 Sensor",
		Location: "Central Park",
		Unit:     "μg/m³",
		Min:      0,
		Max:      500,
	}

	sensors["s2"] = &Sensor{
		ID:       "s2",
		Name:     "CO Sensor",
		Location: "Downtown",
		Unit:     "ppm",
		Min:      0,
		Max:      50,
	}

	// Запуск генерации данных для датчиков
	for _, sensor := range sensors {
		go sensor.generateData()
	}

	// Запуск генерации экстренных событий
	go generateEmergencies()
}

func (s *Sensor) generateData() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		s.value = s.Min + rand.Float64()*(s.Max-s.Min)
		s.mu.Unlock()

		// Отправка данных через UDP
		data, _ := json.Marshal(struct {
			ID    string  `json:"id"`
			Value float64 `json:"value"`
			Unit  string  `json:"unit"`
		}{
			ID:    s.ID,
			Value: s.value,
			Unit:  s.Unit,
		})

		udpAddr, _ := net.ResolveUDPAddr("udp", "localhost:8081")
		conn, _ := net.DialUDP("udp", nil, udpAddr)
		defer conn.Close()
		conn.Write(data)
	}
}

func generateEmergencies() {
	ticker := time.NewTicker(time.Second * 15)
	defer ticker.Stop()

	levels := []string{"low", "medium", "high"}
	messages := []string{
		"Критический уровень загрязнения!",
		"Неисправность оборудования!",
		"Достигнуты экстремальные значения!",
	}

	for range ticker.C {
		if rand.Float32() < 0.9 { // 30% chance every minute
			sensorIDs := make([]string, 0, len(sensors))
			for id := range sensors {
				sensorIDs = append(sensorIDs, id)
			}

			alert := EmergencyAlert{
				SensorID:  sensorIDs[rand.Intn(len(sensorIDs))],
				Message:   messages[rand.Intn(len(messages))],
				Level:     levels[rand.Intn(len(levels))],
				Timestamp: time.Now(),
			}

			sendEmergencyAlert(alert)
		}
	}
}

func sendEmergencyAlert(alert EmergencyAlert) {
	wsClientsMu.Lock()
	defer wsClientsMu.Unlock()

	data, _ := json.Marshal(alert)
	for client := range wsClients {
		err := client.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Printf("WebSocket error: %v", err)
			client.Close()
			delete(wsClients, client)
		}
	}
}

func startTCPServer(port string) {
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("TCP server failed: %v", err)
	}
	defer listener.Close()

	log.Printf("TCP server listening on %s", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("TCP accept error: %v", err)
			continue
		}
		go handleTCPConnection(conn)
	}
}

func handleTCPConnection(conn net.Conn) {
	defer conn.Close()

	// Отправляем список датчиков
	sensorList := make([]Sensor, 0, len(sensors))
	for _, s := range sensors {
		sensorList = append(sensorList, *s)
	}

	data, _ := json.Marshal(sensorList)
	conn.Write(append(data, '\n'))
}

func startUDPServer(port string) {
	udpAddr, _ := net.ResolveUDPAddr("udp", port)
	conn, _ := net.ListenUDP("udp", udpAddr)
	defer conn.Close()

	log.Printf("UDP server listening on %s", port)

	buffer := make([]byte, 1024)
	for {
		_, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("UDP read error: %v", err)
			continue
		}

		// Эхо-ответ для проверки подключения
		conn.WriteToUDP([]byte("ACK"), addr)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	wsClientsMu.Lock()
	wsClients[conn] = true
	wsClientsMu.Unlock()

	// Ожидаем закрытия соединения
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			wsClientsMu.Lock()
			delete(wsClients, conn)
			wsClientsMu.Unlock()
			break
		}
	}
}