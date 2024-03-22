package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type Item struct {
	ItemID      uint   `json:"itemId"`
	ItemCode    string `json:"itemCode"`
	Description string `json:"description"`
	Quantity    uint   `json:"quantity"`
}

type Order struct {
	OrderID      uint      `json:"orderId"`
	CustomerName string    `json:"customerName"`
	OrderedAt    time.Time `json:"orderedAt"`
	Items        []Item    `json:"items"`
}


func main() {
	host := "127.0.0.1"
	port := "5432"
	user := "postgres"
	password := "root"
	dbname := "assignment_2"

	connectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	router := gin.Default()
    router.POST("/orders", func(c *gin.Context) {
        var order Order
        if err := c.BindJSON(&order); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }

        query := `INSERT INTO orders (customer_name, ordered_at) VALUES ($1, $2) RETURNING order_id`
        var orderID int
        err := db.QueryRow(query, order.CustomerName, order.OrderedAt).Scan(&orderID)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }

        for _, item := range order.Items {
            query = `INSERT INTO items (item_code, description, quantity, order_id) VALUES ($1, $2, $3, $4)`
            _, err := db.Exec(query, item.ItemCode, item.Description, item.Quantity, orderID)
            if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
            }
        }
        c.JSON(http.StatusCreated, gin.H{"message": "Order added successfully"})
    })
	
    router.GET("/orders", func(c *gin.Context) {
        rows, err := db.Query("SELECT o.order_id, o.customer_name, o.ordered_at, i.item_id, i.item_code, i.description, i.quantity FROM orders o LEFT JOIN items i ON o.order_id = i.order_id ORDER BY o.order_id")
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        defer rows.Close()

        ordersMap := make(map[uint]*Order)
        for rows.Next() {
            var orderID uint
            var customerName string
            var orderedAt time.Time
            var itemID sql.NullInt64
            var itemCode sql.NullString
            var description sql.NullString
            var quantity sql.NullInt64

            err := rows.Scan(&orderID, &customerName, &orderedAt, &itemID, &itemCode, &description, &quantity)
            if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
            }

            if _, ok := ordersMap[orderID]; !ok {
                ordersMap[orderID] = &Order{
                    OrderID:      orderID,
                    CustomerName: customerName,
                    OrderedAt:    orderedAt,
                    Items:        make([]Item, 0),
                }
            }

            if itemID.Valid {
                ordersMap[orderID].Items = append(ordersMap[orderID].Items, Item{
                    ItemID:      uint(itemID.Int64),
                    ItemCode:    itemCode.String,
                    Description: description.String,
                    Quantity:    uint(quantity.Int64),
                })
            }
        }

        var orders []Order
        for _, order := range ordersMap {
            orders = append(orders, *order)
        }

        c.JSON(http.StatusOK, orders)
    })

    router.PUT("/orders/:orderId", func(c *gin.Context) {
        orderID, err := strconv.Atoi(c.Param("orderId"))
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
            return
        }
        var order Order
        if err := c.BindJSON(&order); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        tx, err := db.Begin()
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        _, err = tx.Exec("UPDATE orders SET customer_name=$1, ordered_at=$2 WHERE order_id=$3", order.CustomerName, order.OrderedAt, orderID)
        if err != nil {
            tx.Rollback()
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        _, err = tx.Exec("DELETE FROM items WHERE order_id=$1", orderID)
        if err != nil {
            tx.Rollback()
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        for _, item := range order.Items {
            _, err = tx.Exec("INSERT INTO items (item_code, description, quantity, order_id) VALUES ($1, $2, $3, $4)", item.ItemCode, item.Description, item.Quantity, orderID)
            if err != nil {
                tx.Rollback()
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
            }
        }
        if err := tx.Commit(); err != nil {
            tx.Rollback()
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusOK, gin.H{"message": "Order updated successfully"})
    })

    router.DELETE("/orders/:orderId", func(c *gin.Context) {
        orderID, err := strconv.Atoi(c.Param("orderId"))
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
            return
        }
        var count int
        err = db.QueryRow("SELECT COUNT(*) FROM orders WHERE order_id = $1", orderID).Scan(&count)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        if count == 0 {
            c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
            return
        }
        tx, err := db.Begin()
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        _, err = tx.Exec("DELETE FROM items WHERE order_id = $1", orderID)
        if err != nil {
            tx.Rollback()
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        _, err = tx.Exec("DELETE FROM orders WHERE order_id = $1", orderID)
        if err != nil {
            tx.Rollback()
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        if err := tx.Commit(); err != nil {
            tx.Rollback()
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusOK, gin.H{"message": "Order deleted successfully"})
    })
	router.Run(":8080")
}