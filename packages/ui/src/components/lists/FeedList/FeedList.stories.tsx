import { Meta, Story } from "@storybook/react";
import { FeedList as Component } from '../';
import { FeedListProps as Props } from '../types';

// Define story metadata
export default {
    title: 'lists/FeedList',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story<Props<any>> = (args) => <Component {...args} />;

// Export story
export const Default = Template.bind({});