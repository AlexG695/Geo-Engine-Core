import { useEffect, useState, useMemo, useRef } from 'react';
import { MapContainer, TileLayer, Marker, Popup, Polyline, Polygon, useMapEvents } from 'react-leaflet';
import axios from 'axios';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';

const styles = `
  @keyframes slideIn { from { opacity: 0; transform: translateX(-20px); } to { opacity: 1; transform: translateX(0); } }
  @keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }
  .notification-card { animation: slideIn 0.3s ease-out forwards; transition: all 0.3s ease; }
  .modal-backdrop { animation: fadeIn 0.2s ease-out forwards; }
  .leaflet-popup-content-wrapper { background: rgba(30, 30, 40, 0.95); color: white; border-radius: 8px; }
  .leaflet-popup-tip { background: rgba(30, 30, 40, 0.95); }
  .leaflet-popup-close-button { color: white !important; }
  /* Cursor para indicar que los puntos se mueven */
  .leaflet-marker-icon.draggable-point { cursor: move !important; }
`;

// --- ICONOS ---
const createCarIcon = (heading: number) => {
    return L.divIcon({
        className: 'car-marker-icon', iconSize: [52, 52], iconAnchor: [26, 26], popupAnchor: [0, -26],
        html: `<div style="transform: rotate(${heading}deg); transition: transform 0.5s cubic-bezier(0.4, 0, 0.2, 1); display: flex; justify-content: center; align-items: center; filter: drop-shadow(0px 4px 6px rgba(0,0,0,0.5)); width: 100%; height: 100%;">
            <svg viewBox="0 0 100 200" width="48" height="96" fill="none" xmlns="http://www.w3.org/2000/svg" style="transform: scale(0.5);">
                <ellipse cx="50" cy="100" rx="45" ry="90" fill="black" fill-opacity="0.3"/>
                <path d="M20 30 Q 10 30, 10 60 L 10 160 Q 10 190, 30 195 L 70 195 Q 90 190, 90 160 L 90 60 Q 90 30, 80 30 L 20 30 Z" fill="#FFFFFF" stroke="#CCCCCC" stroke-width="2"/>
                <path d="M15 65 Q 50 55, 85 65 L 82 90 Q 50 85, 18 90 Z" fill="#2C3E50"/>
                <path d="M20 160 Q 50 165, 80 160 L 78 140 Q 50 145, 22 140 Z" fill="#2C3E50"/>
                <rect x="18" y="90" width="64" height="50" rx="5" fill="#FFFFFF"/>
                <path d="M20 30 Q 25 35, 12 45" stroke="#F1C40F" stroke-width="4" stroke-linecap="round"/>
                <path d="M80 30 Q 75 35, 88 45" stroke="#F1C40F" stroke-width="4" stroke-linecap="round"/>
                <path d="M20 192 L 10 185" stroke="#E74C3C" stroke-width="4" stroke-linecap="round"/>
                <path d="M80 192 L 90 185" stroke="#E74C3C" stroke-width="4" stroke-linecap="round"/>
            </svg>
        </div>`
    });
};

const dotIcon = new L.DivIcon({
    className: 'draggable-point', // Clase CSS para cursor
    html: '<div style="width: 14px; height: 14px; background-color: #f59e0b; border-radius: 50%; border: 2px solid white; box-shadow: 0 0 4px rgba(0,0,0,0.5);"></div>',
    iconSize: [14, 14], iconAnchor: [7, 7],
});

// --- COMPONENTE DE PUNTO ARRASTRABLE ---
// Esto permite mover los puntos individuales sin romper la geometría
const DraggableMarker = ({ position, index, onDrag, onClick }: { position: [number, number], index: number, onDrag: (i: number, lat: number, lng: number) => void, onClick: () => void }) => {
    const markerRef = useRef<L.Marker>(null);

    const eventHandlers = useMemo(() => ({
        dragend() {
            const marker = markerRef.current;
            if (marker != null) {
                const { lat, lng } = marker.getLatLng();
                onDrag(index, lat, lng);
            }
        },
        click(e: any) {
            L.DomEvent.stopPropagation(e); // ¡CRUCIAL! Evita que el mapa detecte click y agregue otro punto
            onClick();
        },
        mousedown(e: any) {
            L.DomEvent.stopPropagation(e); // Evita conflictos de drag con el mapa
        }
    }), [index, onDrag, onClick]);

    return (
        <Marker
            draggable={true} // Hacemos que sea movible
            eventHandlers={eventHandlers}
            position={position}
            icon={dotIcon}
            ref={markerRef}
        />
    );
};

const MapClickHandler = ({ isDrawing, onAddPoint, onClearSelection }: any) => {
    useMapEvents({
        click(e) {
            if (isDrawing) onAddPoint([e.latlng.lat, e.latlng.lng]);
            else onClearSelection();
        },
    });
    return null;
};

interface Driver { id: string; device_id: string; latitude: number; longitude: number; heading: number; }
interface Alert { id: number; title: string; body: string; time: string; color: string; bg: string; icon: string; }
interface Toast { msg: string; type: 'success' | 'error'; }
interface ModalConfig { show: boolean; type: 'create' | 'rename' | 'delete' | null; id?: string; initialValue?: string; }

function App() {
    const [drivers, setDrivers] = useState<Driver[]>([]);
    const [selectedRoute, setSelectedRoute] = useState<[number, number][]>([]);
    const [selectedId, setSelectedId] = useState<string | null>(null);
    const [geofences, setGeofences] = useState<any[]>([]);
    const [alerts, setAlerts] = useState<Alert[]>([]);

    // UI States
    const [isDrawing, setIsDrawing] = useState(false);
    const [editingId, setEditingId] = useState<string | null>(null);
    const [newPolygon, setNewPolygon] = useState<[number, number][]>([]);

    const [modal, setModal] = useState<ModalConfig>({ show: false, type: null });
    const [modalInput, setModalInput] = useState('');
    const [toast, setToast] = useState<Toast | null>(null);

    const centerPos: [number, number] = [28.6353, -106.0889];

    const showToast = (msg: string, type: 'success' | 'error') => {
        setToast({ msg, type });
        setTimeout(() => setToast(null), 3000);
    };

    // --- API ---
    const fetchDrivers = async () => {
        try {
            const res = await axios.get('http://localhost:8080/drivers/nearby', { params: { lat: centerPos[0], lng: centerPos[1], radius: 50000 } });
            setDrivers(res.data.data || []);
        } catch (error) { console.error(error); }
    };

    const fetchRoute = async (deviceId: string) => {
        try {
            setSelectedId(deviceId);
            const res = await axios.get(`http://localhost:8080/drivers/${deviceId}/route`);
            const coords = res.data.coordinates.map((c: number[]) => [c[1], c[0]] as [number, number]);
            setSelectedRoute(coords);
        } catch (e) { setSelectedRoute([]); }
    };

    const fetchGeofences = async () => {
        try {
            const res = await axios.get('http://localhost:8080/geofences');
            const parsed = res.data.map((z: any) => ({
                ...z,
                coordinates: JSON.parse(z.geojson).coordinates[0].map((c: number[]) => [c[1], c[0]])
            }));
            setGeofences(parsed);
        } catch (e) { console.error(e); }
    };

    // --- MANEJO DE GEOMETRÍA ---

    // Actualizar posición de un punto específico al arrastrar
    const handleDragPoint = (index: number, lat: number, lng: number) => {
        setNewPolygon(prev => {
            const updated = [...prev];
            updated[index] = [lat, lng];
            return updated;
        });
    };

    // Eliminar un punto con click (Opcional, útil para limpiar)
    const handlePointClick = (index: number) => {
        if (newPolygon.length <= 3) {
            showToast("Mínimo 3 puntos requeridos", "error");
            return;
        }
        setNewPolygon(prev => prev.filter((_, i) => i !== index));
    };

    const handleSaveClick = () => {
        if (newPolygon.length < 3) { showToast("Necesitas al menos 3 puntos", "error"); return; }

        if (editingId) {
            const current = geofences.find(g => g.id === editingId);
            setModalInput(current?.name || '');
            setModal({ show: true, type: 'create', id: editingId });
        } else {
            setModalInput('');
            setModal({ show: true, type: 'create' });
        }
    };

    const handleModalConfirm = async () => {
        if (modal.type === 'delete' && modal.id) {
            try {
                await axios.delete(`http://localhost:8080/geofences/${modal.id}`);
                showToast("Zona eliminada", "success");
                setModal({ show: false, type: null });
                fetchGeofences();
            } catch (e) { showToast("Error al eliminar", "error"); }
        }
        else if ((modal.type === 'create' || modal.type === 'rename') && modalInput.trim()) {
            try {
                let geojsonPayload = null;
                if (modal.type === 'create' || editingId) {
                    const coordinates = newPolygon.map(p => [p[1], p[0]]);
                    coordinates.push(coordinates[0]);
                    geojsonPayload = JSON.stringify({ type: "Polygon", coordinates: [coordinates] });
                }

                if (modal.type === 'create' && !editingId) {
                    await axios.post('http://localhost:8080/geofences', { name: modalInput, geojson: geojsonPayload });
                    showToast("Zona creada", "success");
                    setNewPolygon([]); setIsDrawing(false);
                } else {
                    const idToUpdate = modal.id || editingId;
                    const payload: any = { name: modalInput };
                    if (geojsonPayload) payload.geojson = geojsonPayload;

                    await axios.put(`http://localhost:8080/geofences/${idToUpdate}`, payload);
                    showToast("Zona actualizada", "success");
                    setNewPolygon([]); setIsDrawing(false); setEditingId(null);
                }

                setModal({ show: false, type: null });
                fetchGeofences();
            } catch (e) { showToast("Error al guardar", "error"); }
        }
    };

    const startEditGeometry = (geo: any) => {
        setEditingId(geo.id);
        setNewPolygon(geo.coordinates);
        setIsDrawing(true);
    };

    const openRenameModal = (geo: any) => {
        setModalInput(geo.name);
        setModal({ show: true, type: 'rename', id: geo.id });
    };

    const openDeleteModal = (geo: any) => {
        setModal({ show: true, type: 'delete', id: geo.id, initialValue: geo.name });
    };

    useEffect(() => {
        fetchDrivers(); fetchGeofences();
        const socket = new WebSocket("ws://localhost:8080/ws");
        socket.onmessage = (event) => {
            try {
                const msg = JSON.parse(event.data);
                if (msg.type === "LOCATION_UPDATE") {
                    setDrivers(prev => {
                        const exists = prev.find(d => d.device_id === msg.device_id);
                        if (exists) return prev.map(d => d.device_id === msg.device_id ? { ...d, latitude: msg.latitude, longitude: msg.longitude, heading: msg.heading } : d);
                        else return [...prev, { id: "live-" + msg.device_id, device_id: msg.device_id, latitude: msg.latitude, longitude: msg.longitude, heading: msg.heading }];
                    });
                } else if (msg.type === "GEOFENCE_EVENT") {
                    const isEnter = msg.event === "ENTER";
                    const newAlert: Alert = {
                        id: Date.now(), title: isEnter ? "ENTRADA" : "SALIDA", body: `${msg.device_id} en ${msg.zone_name}`,
                        time: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' }),
                        color: isEnter ? "#00E676" : "#FF5252", icon: isEnter ? "🛡️" : "⚠️", bg: isEnter ? "rgba(0, 230, 118, 0.1)" : "rgba(255, 82, 82, 0.1)"
                    };
                    setAlerts(prev => [newAlert, ...prev].slice(0, 4));
                }
            } catch (error) { console.error(error); }
        };
        return () => { socket.close(); };
    }, []);

    return (
        <div style={{ height: '100vh', width: '100vw', overflow: 'hidden' }}>
            <style>{styles}</style>

            <MapContainer center={centerPos} zoom={14} style={{ height: '100%', width: '100%' }}>
                <TileLayer url="https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png" attribution='&copy; CARTO'/>

                <MapClickHandler isDrawing={isDrawing} onAddPoint={(p: any) => setNewPolygon(prev => [...prev, p])} onClearSelection={() => { setSelectedRoute([]); setSelectedId(null); }} />

                {/* --- MODO DIBUJO / EDICIÓN --- */}
                {newPolygon.length > 0 && (
                    <>
                        {/* Usamos Polyline mientras dibujamos para evitar efecto visual de cierre prematuro */}
                        <Polyline positions={newPolygon} pathOptions={{ color: '#f59e0b', dashArray: '5, 10', weight: 2 }} />
                        {/* Línea de cierre sugerida (visual) */}
                        {newPolygon.length > 2 && <Polyline positions={[newPolygon[newPolygon.length -1], newPolygon[0]]} pathOptions={{ color: '#f59e0b', dashArray: '2, 5', weight: 1, opacity: 0.5 }} />}

                        {/* PUNTOS ARRASTRABLES */}
                        {newPolygon.map((pos, idx) => (
                            <DraggableMarker
                                key={`p-${idx}`}
                                index={idx}
                                position={pos}
                                onDrag={handleDragPoint}
                                onClick={() => handlePointClick(idx)}
                            />
                        ))}
                    </>
                )}

                {/* --- ZONAS GUARDADAS --- */}
                {geofences.map((geo) => {
                    if (geo.id === editingId) return null; // Ocultar si se está editando
                    return (
                        <Polygon key={geo.id} positions={geo.coordinates} pathOptions={{ color: '#FF5252', fillOpacity: 0.15, weight: 2 }}>
                            <Popup>
                                <div style={{ textAlign: 'center', minWidth: '150px' }}>
                                    <h3 style={{ margin: '0 0 10px 0', fontSize: '15px', fontWeight: 'bold' }}>{geo.name}</h3>
                                    <div style={{ display: 'grid', gap: '8px', gridTemplateColumns: '1fr 1fr' }}>
                                        <button onClick={(e) => { e.stopPropagation(); startEditGeometry(geo); }} style={{ background: '#f59e0b', border: 'none', borderRadius: '4px', padding: '6px', color: 'white', cursor: 'pointer', fontSize: '11px', gridColumn: '1 / -1' }}>
                                            📐 Editar Forma
                                        </button>
                                        <button onClick={(e) => { e.stopPropagation(); openRenameModal(geo); }} style={{ background: '#3b82f6', border: 'none', borderRadius: '4px', padding: '6px', color: 'white', cursor: 'pointer', fontSize: '11px' }}>
                                            ✏️ Nombre
                                        </button>
                                        <button onClick={(e) => { e.stopPropagation(); openDeleteModal(geo); }} style={{ background: '#ef4444', border: 'none', borderRadius: '4px', padding: '6px', color: 'white', cursor: 'pointer', fontSize: '11px' }}>
                                            🗑️ Borrar
                                        </button>
                                    </div>
                                </div>
                            </Popup>
                        </Polygon>
                    );
                })}

                {/* Ruta y Autos */}
                {selectedRoute.length > 0 && <Polyline positions={selectedRoute} pathOptions={{ color: '#F1C40F', weight: 5, opacity: 0.8, lineCap: 'round' }} />}
                {drivers.map((driver) => (
                    <Marker key={driver.device_id} position={[driver.latitude, driver.longitude]} icon={createCarIcon(driver.heading || 0)} opacity={selectedId && selectedId !== driver.device_id ? 0.3 : 1} eventHandlers={{ click: (e) => { L.DomEvent.stopPropagation(e); fetchRoute(driver.device_id); } }}>
                        <Popup>ID: {driver.device_id}</Popup>
                    </Marker>
                ))}
            </MapContainer>

            {/* UI ELEMENTS */}
            {toast && (
                <div className="toast-message" style={{ position: 'absolute', top: 20, left: '50%', transform: 'translateX(-50%)', zIndex: 2000, background: toast.type === 'success' ? 'rgba(0, 230, 118, 0.9)' : 'rgba(255, 82, 82, 0.9)', color: 'white', padding: '12px 24px', borderRadius: '50px', fontWeight: 'bold', boxShadow: '0 4px 15px rgba(0,0,0,0.3)', display: 'flex', alignItems: 'center', gap: '10px', backdropFilter: 'blur(5px)' }}>
                    {toast.type === 'success' ? '✅' : '❌'} {toast.msg}
                </div>
            )}

            <div style={{ position: 'absolute', top: 20, right: 60, zIndex: 1000, background: 'rgba(30, 30, 30, 0.9)', color: 'white', padding: '15px', borderRadius: '8px', backdropFilter: 'blur(5px)' }}>
                <h3 style={{margin: '0 0 10px 0'}}>Geo Engine 🚀</h3>
                <p style={{margin: '0 0 5px 0'}}>📡 Conexión: <span style={{color: '#2ECC71'}}>Activa</span></p>
                <p style={{margin: 0}}>Conductores: <strong>{drivers.length}</strong> | Zonas: <strong>{geofences.length}</strong></p>
            </div>

            <div style={{ position: 'absolute', bottom: 30, right: 30, zIndex: 1000, display: 'flex', gap: '10px' }}>
                {!isDrawing ? (
                    <button onClick={() => setIsDrawing(true)} style={{ padding: '12px 24px', background: '#3b82f6', color: 'white', border: 'none', borderRadius: '50px', fontWeight: 'bold', boxShadow: '0 4px 12px rgba(0,0,0,0.3)', cursor: 'pointer', transition: 'transform 0.2s' }} onMouseOver={(e) => e.currentTarget.style.transform = 'scale(1.05)'} onMouseOut={(e) => e.currentTarget.style.transform = 'scale(1)'}>
                        ✏️ Nueva Zona
                    </button>
                ) : (
                    <>
                        <button onClick={handleSaveClick} style={{ padding: '12px 24px', background: '#10b981', color: 'white', border: 'none', borderRadius: '50px', fontWeight: 'bold', boxShadow: '0 4px 12px rgba(0,0,0,0.3)', cursor: 'pointer' }}>💾 Guardar</button>
                        <button onClick={() => { setIsDrawing(false); setNewPolygon([]); setEditingId(null); }} style={{ padding: '12px 24px', background: '#ef4444', color: 'white', border: 'none', borderRadius: '50px', fontWeight: 'bold', boxShadow: '0 4px 12px rgba(0,0,0,0.3)', cursor: 'pointer' }}>❌ Cancelar</button>
                    </>
                )}
            </div>

            {isDrawing && (
                <div style={{ position: 'absolute', top: 80, left: '50%', transform: 'translateX(-50%)', zIndex: 1000, background: 'rgba(0,0,0,0.8)', color: 'white', padding: '10px 20px', borderRadius: '20px', pointerEvents: 'none' }}>
                    {editingId ? '📐 Arrastra los puntos para ajustar' : '📍 Haz clic para añadir puntos'}
                </div>
            )}

            {/* MODAL */}
            {modal.show && (
                <div className="modal-backdrop" style={{ position: 'absolute', inset: 0, zIndex: 2000, background: 'rgba(0,0,0,0.6)', backdropFilter: 'blur(3px)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                    <div style={{ background: 'rgba(30, 30, 40, 0.95)', padding: '30px', borderRadius: '16px', width: '350px', border: '1px solid rgba(255,255,255,0.1)', boxShadow: '0 10px 40px rgba(0,0,0,0.5)', color: 'white', textAlign: 'center' }}>
                        <h3 style={{ margin: '0 0 15px 0', fontSize: '20px' }}>{modal.type === 'delete' ? 'Eliminar Zona' : modal.type === 'rename' ? 'Renombrar Zona' : 'Guardar Zona'}</h3>
                        {modal.type === 'delete' ? (
                            <p style={{ color: '#aaa', marginBottom: '25px' }}>¿Eliminar <strong>{modal.initialValue}</strong>?</p>
                        ) : (
                            <input autoFocus type="text" value={modalInput} onChange={(e) => setModalInput(e.target.value)} placeholder="Nombre" style={{ width: '100%', padding: '12px', borderRadius: '8px', background: 'rgba(0,0,0,0.3)', border: '1px solid #555', color: 'white', fontSize: '16px', marginBottom: '25px', outline: 'none' }} onKeyDown={(e) => e.key === 'Enter' && handleModalConfirm()} />
                        )}
                        <div style={{ display: 'flex', gap: '10px', justifyContent: 'center' }}>
                            <button onClick={() => setModal({ show: false, type: null })} style={{ padding: '10px 20px', borderRadius: '8px', border: 'none', background: 'transparent', color: '#aaa', cursor: 'pointer' }}>Cancelar</button>
                            <button onClick={handleModalConfirm} disabled={modal.type !== 'delete' && !modalInput.trim()} style={{ padding: '10px 20px', borderRadius: '8px', border: 'none', background: modal.type === 'delete' ? '#ef4444' : '#3b82f6', color: 'white', cursor: (modal.type === 'delete' || modalInput.trim()) ? 'pointer' : 'not-allowed', fontWeight: 'bold' }}>{modal.type === 'delete' ? 'Eliminar' : 'Confirmar'}</button>
                        </div>
                    </div>
                </div>
            )}

            <div style={{ position: 'absolute', bottom: 30, left: 20, zIndex: 1000, display: 'flex', flexDirection: 'column', gap: '10px', width: '320px', pointerEvents: 'none' }}>
                {alerts.map((alert) => (
                    <div key={alert.id} className="notification-card" style={{ pointerEvents: 'auto', background: 'rgba(20, 20, 30, 0.85)', backdropFilter: 'blur(12px)', borderLeft: `4px solid ${alert.color}`, borderRadius: '0 8px 8px 0', padding: '12px 16px', color: 'white', boxShadow: '0 4px 15px rgba(0,0,0,0.4)', display: 'flex', alignItems: 'center', gap: '12px', borderTop: '1px solid rgba(255,255,255,0.1)' }}>
                        <div style={{ fontSize: '24px', background: alert.bg, width: '40px', height: '40px', display: 'flex', alignItems: 'center', justifyContent: 'center', borderRadius: '50%' }}>{alert.icon}</div>
                        <div style={{ flex: 1 }}>
                            <div style={{ fontSize: '10px', textTransform: 'uppercase', color: alert.color, fontWeight: 'bold' }}>{alert.title}</div>
                            <div style={{ fontSize: '14px', fontWeight: 500 }}>{alert.body}</div>
                            <div style={{ fontSize: '10px', color: '#aaa' }}>🕒 {alert.time}</div>
                        </div>
                        <button onClick={() => setAlerts(p => p.filter(a => a.id !== alert.id))} style={{ background: 'transparent', border: 'none', color: '#666', cursor: 'pointer', fontSize: '16px' }}>×</button>
                    </div>
                ))}
            </div>
        </div>
    );
}

export default App;