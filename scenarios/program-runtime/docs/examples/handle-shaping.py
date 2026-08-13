"""Shape governed rows in the kernel without materializing the source handles."""


orders = Handle(
    [
        {"customer_id": "c-1", "status": "paid", "amount": 24},
        {"customer_id": "c-2", "status": "paid", "amount": 31},
        {"customer_id": "c-1", "status": "paid", "amount": 18},
        {"customer_id": "c-3", "status": "pending", "amount": 9},
    ],
    "orders",
)
customers = Handle(
    [
        {"customer_id": "c-1", "name": "Ada"},
        {"customer_id": "c-2", "name": "Grace"},
        {"customer_id": "c-3", "name": "Edsger"},
    ],
    "customers",
)

paid = orders.filter(lambda row: row["status"] == "paid")
shaped = (
    paid.join(customers, "customer_id")
    .sort("amount", reverse=True)
    .select("name", "amount")
)

print(shaped.head(3))
print("paid_total=", paid.agg("amount", "sum"))
