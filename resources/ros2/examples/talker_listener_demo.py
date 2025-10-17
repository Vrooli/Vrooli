#!/usr/bin/env python3

"""
ROS2 Talker/Listener Demo
Demonstrates basic pub/sub communication pattern
"""

import sys
import time
import threading

# Simulated ROS2 functionality for demonstration
# In real implementation, would use actual rclpy

class SimulatedNode:
    """Simulated ROS2 node for demonstration"""
    
    def __init__(self, name):
        self.name = name
        self.publishers = {}
        self.subscribers = {}
        self.running = False
        
    def create_publisher(self, topic, msg_type):
        """Create a publisher for a topic"""
        self.publishers[topic] = {
            'type': msg_type,
            'subscribers': []
        }
        print(f"[{self.name}] Created publisher on topic: {topic}")
        return topic
        
    def create_subscriber(self, topic, msg_type, callback):
        """Create a subscriber for a topic"""
        self.subscribers[topic] = {
            'type': msg_type,
            'callback': callback
        }
        print(f"[{self.name}] Created subscriber on topic: {topic}")
        
    def publish(self, topic, message):
        """Publish a message to a topic"""
        if topic in self.publishers:
            print(f"[{self.name}] Publishing to {topic}: {message}")
            # In real implementation, would send via DDS
            
    def spin(self):
        """Keep node running"""
        self.running = True
        while self.running:
            time.sleep(0.1)
            
    def shutdown(self):
        """Shutdown the node"""
        self.running = False
        print(f"[{self.name}] Shutting down")

class TalkerNode(SimulatedNode):
    """Talker node that publishes messages"""
    
    def __init__(self):
        super().__init__('talker')
        self.topic = self.create_publisher('/chatter', 'std_msgs/String')
        self.counter = 0
        
    def run(self):
        """Main talker loop"""
        self.running = True
        while self.running:
            message = f"Hello World: {self.counter}"
            self.publish(self.topic, message)
            self.counter += 1
            time.sleep(1)

class ListenerNode(SimulatedNode):
    """Listener node that receives messages"""
    
    def __init__(self):
        super().__init__('listener')
        self.create_subscriber('/chatter', 'std_msgs/String', self.callback)
        self.messages_received = 0
        
    def callback(self, message):
        """Handle received messages"""
        print(f"[{self.name}] Received: {message}")
        self.messages_received += 1
        
    def run(self):
        """Main listener loop"""
        self.spin()

def main():
    """Run the talker/listener demo"""
    print("ROS2 Talker/Listener Demo")
    print("=" * 40)
    
    # Create nodes
    talker = TalkerNode()
    listener = ListenerNode()
    
    # Start nodes in threads
    talker_thread = threading.Thread(target=talker.run)
    listener_thread = threading.Thread(target=listener.run)
    
    talker_thread.start()
    listener_thread.start()
    
    try:
        # Run for 5 seconds
        time.sleep(5)
    except KeyboardInterrupt:
        print("\nInterrupted by user")
    
    # Shutdown nodes
    print("\nShutting down nodes...")
    talker.shutdown()
    listener.shutdown()
    
    talker_thread.join(timeout=1)
    listener_thread.join(timeout=1)
    
    print(f"\nDemo complete!")
    print(f"Talker sent {talker.counter} messages")
    print(f"Listener received {listener.messages_received} messages")

if __name__ == '__main__':
    main()