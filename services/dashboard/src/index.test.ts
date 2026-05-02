import request from "supertest";
import { app } from "./index";

describe("Dashboard API", () => {
  describe("GET /health", () => {
    it("should return healthy status", async () => {
      const res = await request(app).get("/health");
      expect(res.status).toBe(200);
      expect(res.body.status).toBe("healthy");
      expect(res.body.service).toBe("dashboard");
      expect(res.body.timestamp).toBeDefined();
    });
  });

  describe("GET /stats", () => {
    it("should return empty stats initially", async () => {
      const res = await request(app).get("/stats");
      expect(res.status).toBe(200);
      expect(res.body.total_pipelines).toBe(0);
      expect(res.body.total_metrics).toBe(0);
      expect(res.body.active_alerts).toBe(0);
    });
  });

  describe("POST /metrics", () => {
    it("should record a metric", async () => {
      const res = await request(app).post("/metrics").send({
        pipeline_id: "p1",
        metric_name: "throughput",
        value: 1000,
      });
      expect(res.status).toBe(201);
      expect(res.body.pipeline_id).toBe("p1");
      expect(res.body.metric_name).toBe("throughput");
      expect(res.body.value).toBe(1000);
    });

    it("should reject metric without required fields", async () => {
      const res = await request(app).post("/metrics").send({
        pipeline_id: "p1",
      });
      expect(res.status).toBe(400);
    });

    it("should create alert for high error rate", async () => {
      await request(app).post("/metrics").send({
        pipeline_id: "p2",
        metric_name: "error_rate",
        value: 0.9,
      });

      const alertRes = await request(app).get("/alerts?resolved=false");
      expect(alertRes.status).toBe(200);
      const critAlerts = alertRes.body.filter(
        (a: { severity: string }) => a.severity === "critical"
      );
      expect(critAlerts.length).toBeGreaterThan(0);
    });
  });

  describe("GET /metrics", () => {
    it("should list all metrics", async () => {
      const res = await request(app).get("/metrics");
      expect(res.status).toBe(200);
      expect(Array.isArray(res.body)).toBe(true);
    });

    it("should filter by pipeline_id", async () => {
      const res = await request(app).get("/metrics?pipeline_id=p1");
      expect(res.status).toBe(200);
      for (const m of res.body) {
        expect(m.pipeline_id).toBe("p1");
      }
    });
  });

  describe("GET /alerts", () => {
    it("should list alerts", async () => {
      const res = await request(app).get("/alerts");
      expect(res.status).toBe(200);
      expect(Array.isArray(res.body)).toBe(true);
    });
  });

  describe("PATCH /alerts/:id/resolve", () => {
    it("should resolve an alert", async () => {
      const alertsRes = await request(app).get("/alerts?resolved=false");
      if (alertsRes.body.length > 0) {
        const alertId = alertsRes.body[0].id;
        const res = await request(app).patch(`/alerts/${alertId}/resolve`);
        expect(res.status).toBe(200);
        expect(res.body.resolved).toBe(true);
      }
    });

    it("should return 404 for non-existent alert", async () => {
      const res = await request(app).patch("/alerts/nonexistent/resolve");
      expect(res.status).toBe(404);
    });
  });
});
