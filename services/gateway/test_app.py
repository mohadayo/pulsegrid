import httpx
import pytest
import respx
from fastapi.testclient import TestClient
from app import app, PROCESSOR_URL, DASHBOARD_URL


client = TestClient(app)


def test_health():
    resp = client.get("/health")
    assert resp.status_code == 200
    data = resp.json()
    assert data["status"] == "healthy"
    assert data["service"] == "gateway"
    assert "timestamp" in data


def test_health_all_with_services_down():
    resp = client.get("/health/all")
    assert resp.status_code == 200
    data = resp.json()
    assert data["services"]["gateway"] == "healthy"
    assert data["services"]["processor"] in ("healthy", "unreachable")
    assert data["services"]["dashboard"] in ("healthy", "unreachable")


def test_create_pipeline_processor_down():
    resp = client.post(
        "/pipelines",
        json={"name": "test-pipe", "source": "kafka", "destination": "s3"},
    )
    assert resp.status_code == 502


def test_list_pipelines_processor_down():
    resp = client.get("/pipelines")
    assert resp.status_code == 502


def test_get_pipeline_processor_down():
    resp = client.get("/pipelines/abc123")
    assert resp.status_code == 502


def test_send_event_processor_down():
    resp = client.post(
        "/pipelines/abc123/events",
        json={
            "pipeline_id": "abc123",
            "event_type": "data_ingested",
            "payload": {"rows": 100},
        },
    )
    assert resp.status_code == 502


def test_dashboard_stats_down():
    resp = client.get("/dashboard/stats")
    assert resp.status_code == 502


@respx.mock
def test_create_pipeline_success():
    respx.post(f"{PROCESSOR_URL}/pipelines").mock(
        return_value=httpx.Response(
            201, json={"id": "p1", "name": "test", "status": "active"}
        )
    )
    resp = client.post(
        "/pipelines",
        json={"name": "test", "source": "kafka", "destination": "s3"},
    )
    assert resp.status_code == 201
    assert resp.json()["id"] == "p1"


@respx.mock
def test_list_pipelines_success():
    respx.get(f"{PROCESSOR_URL}/pipelines").mock(
        return_value=httpx.Response(200, json=[{"id": "p1", "name": "test"}])
    )
    resp = client.get("/pipelines")
    assert resp.status_code == 200
    assert len(resp.json()) == 1


@respx.mock
def test_get_pipeline_success():
    respx.get(f"{PROCESSOR_URL}/pipelines/p1").mock(
        return_value=httpx.Response(
            200, json={"id": "p1", "name": "test", "status": "active"}
        )
    )
    resp = client.get("/pipelines/p1")
    assert resp.status_code == 200
    assert resp.json()["id"] == "p1"


@respx.mock
def test_dashboard_stats_success():
    respx.get(f"{DASHBOARD_URL}/stats").mock(
        return_value=httpx.Response(
            200, json={"total_pipelines": 5, "total_metrics": 100}
        )
    )
    resp = client.get("/dashboard/stats")
    assert resp.status_code == 200
    assert resp.json()["total_pipelines"] == 5
