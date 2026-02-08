import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import App from './App';
import axios from 'axios';

vi.mock('axios');
const mockedAxios = axios as jest.Mocked<typeof axios>;


vi.mock('react-leaflet', () => ({
    MapContainer: ({ children }: any) => <div data-testid="map-container">{children}</div>,
    TileLayer: () => <div>TileLayer</div>,
    Marker: ({ children }: any) => <div data-testid="marker">{children}</div>,
    Popup: ({ children }: any) => <div>Popup {children}</div>,
    Polygon: ({ children }: any) => <div data-testid="polygon">{children}</div>,
    Polyline: () => <div>Polyline</div>,
    useMapEvents: () => null,
}));

// Mock del WebSocket Global
class MockWebSocket {
    onopen: () => void = () => {};
    onmessage: (e: any) => void = () => {};
    close: () => void = () => {};
    constructor() { }
}
global.WebSocket = MockWebSocket as any;

// GeoJSON válido de prueba
const MOCK_GEOJSON = JSON.stringify({
    type: "Polygon",
    coordinates: [[[0,0], [0,1], [1,1], [0,0]]]
});


describe('Geo Engine Frontend', () => {

    beforeEach(() => {
        vi.clearAllMocks();
        mockedAxios.get.mockResolvedValue({ data: [] });
    });

    it('debe renderizar el título y el mapa correctamente', async () => {
        render(<App />);

        await waitFor(() => {
            expect(screen.getByText(/Geo Engine/i)).toBeInTheDocument();
            expect(screen.getByTestId('map-container')).toBeInTheDocument();
        });
    });

    it('debe mostrar las estadísticas iniciales y cargar datos', async () => {
        mockedAxios.get.mockImplementation((url) => {
            if (url.includes('/drivers')) {
                return Promise.resolve({
                    data: {
                        data: [
                            { device_id: 'd1', latitude: 10, longitude: 10, heading: 0 },
                            { device_id: 'd2', latitude: 20, longitude: 20, heading: 0 }
                        ]
                    }
                });
            }
            if (url.includes('/geofences')) {
                return Promise.resolve({
                    data: [
                        { id: 'z1', name: 'Zona Norte', geojson: MOCK_GEOJSON },
                        { id: 'z2', name: 'Zona Sur', geojson: MOCK_GEOJSON },
                        { id: 'z3', name: 'Centro', geojson: MOCK_GEOJSON }
                    ]
                });
            }
            return Promise.resolve({ data: [] });
        });

        render(<App />);

        await waitFor(() => {
            expect(screen.getByText(/Conductores:/i)).toHaveTextContent('2');
            expect(screen.getByText(/Zonas:/i)).toHaveTextContent('3');
        });
    });

    it('debe activar el modo dibujo al hacer clic en "Nueva Zona"', async () => {
        render(<App />);

        // Esperar render inicial
        await waitFor(() => expect(screen.getByText(/Nueva Zona/i)).toBeInTheDocument());

        const drawBtn = screen.getByText(/Nueva Zona/i);
        fireEvent.click(drawBtn);

        await waitFor(() => {
            expect(screen.getByText(/Guardar/i)).toBeInTheDocument();
            expect(screen.getByText(/Cancelar/i)).toBeInTheDocument();
            expect(screen.getByText(/Haz clic para añadir puntos/i)).toBeInTheDocument();
        });
    });

    it('debe mostrar error (toast) al intentar guardar sin puntos suficientes', async () => {
        render(<App />);

        await waitFor(() => expect(screen.getByText(/Nueva Zona/i)).toBeInTheDocument());

        fireEvent.click(screen.getByText(/Nueva Zona/i));

        const saveBtn = screen.getByText(/Guardar/i);
        fireEvent.click(saveBtn);

        await waitFor(() => {
            expect(screen.getByText(/Necesitas al menos 3 puntos/i)).toBeInTheDocument();
        });

        fireEvent.click(screen.getByText(/Cancelar/i));
        await waitFor(() => {
            expect(screen.getByText(/Nueva Zona/i)).toBeInTheDocument();
        });
    });
});